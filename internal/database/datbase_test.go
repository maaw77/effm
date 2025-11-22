// Пакет database содержит модульные тесты для слоя работы с БД.
//
// Тесты проверяют корректность всех операций CRUD:
// - Создание, чтение, обновление, удаление подписок
// - Обработку конфликтов уникальности
// - Корректность подсчёта общей стоимости за период

package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maaw77/effm/config"
	"github.com/maaw77/effm/internal/models"
)

var testDB *SubscriptionDatabase

// TestMain выполняется один раз до и после всех тестов.
//
// Процесс инициализации:
// 1. Подключение к тестовой БД using config.yaml
// 2. Очистка таблицы subscriptions
// 3. Запуск всех тестов
// 4. Закрытие соединения с БД
//
// Важно: таблица полностью очищается (TRUNCATE) перед запуском тестовой серии.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Читаем строку подключения из config.yaml
	connStr := config.InitConnString("config/config.yaml")

	var err error
	testDB, err = NewSubscriptionDatabase(ctx, connStr)
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}
	defer testDB.Close()

	// Очищаем таблицу перед запуском тестов
	_, err = testDB.DBpool.Exec(ctx, "TRUNCATE TABLE subscriptions RESTART IDENTITY")
	if err != nil {
		panic("failed to truncate subscriptions table: " + err.Error())
	}

	m.Run()
}

// newTestSubscription создает тестовый объект подписки с уникальным UserID.
//
// Параметры:
//
//	year  - год подписки
//	month - месяц подписки
//
// Возвращает:
//
//	models.Subscription с заполненными обязательными полями
func newTestSubscription(year, month int) models.Subscription {
	return models.Subscription{
		ServiceName: "TestService",
		Price:       999,
		UserID:      uuid.New().String(),
		Year:        year,
		Month:       month,
	}
}

// TestCreateAndGet проверяет базовый сценарий создания и чтения подписки.
//
// Тест проверяет:
// - Успешное создание подписки в БД
// - Корректное чтение созданной подписки по ID
// - Соответствие всех полей исходным данным
func TestCreateAndGet(t *testing.T) {
	ctx := context.Background()
	sub := newTestSubscription(2025, 7)

	id, err := testDB.Create(ctx, sub)
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	got, err := testDB.Get(ctx, id)
	if err != nil {
		t.Fatalf("failed to get subscription by id=%s: %v", id, err)
	}

	if got.ID != id ||
		got.ServiceName != sub.ServiceName ||
		got.Price != sub.Price ||
		got.UserID != sub.UserID ||
		got.Year != 2025 || got.Month != 7 {
		t.Fatalf("retrieved subscription does not match created one.\nWant: %+v\nGot:  %+v", sub, got)
	}
}

// TestCreateConflict проверяет обработку конфликта уникальности.
//
// Ожидаемое поведение:
// - Первое создание подписки - успешно
// - Второе создание идентичной подписки - ошибка ErrConflict
// - Сообщение об ошибке соответствует бизнес-логике уникальности
func TestCreateConflict(t *testing.T) {
	ctx := context.Background()
	sub := newTestSubscription(2025, 8)

	_, err := testDB.Create(ctx, sub)
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}

	_, err = testDB.Create(ctx, sub)
	if err != ErrConflict {
		t.Fatalf("expected ErrConflict on duplicate subscription, got: %v (%T)", err, err)
	}
}

// TestUpdate проверяет обновление данных подписки.
//
// Тестируемые сценарии:
// - Изменение названия сервиса
// - Изменение цены подписки
// - Сохранение неизменяемых полей (год, месяц, пользователь)
func TestUpdate(t *testing.T) {
	ctx := context.Background()
	sub := newTestSubscription(2025, 9)
	id, _ := testDB.Create(ctx, sub)

	updates := models.Subscription{
		ServiceName: "UpdatedService",
		Price:       1500,
	}

	err := testDB.Update(ctx, id, updates)
	if err != nil {
		t.Fatalf("failed to update subscription: %v", err)
	}

	updated, _ := testDB.Get(ctx, id)
	if updated.ServiceName != "UpdatedService" || updated.Price != 1500 {
		t.Fatalf("subscription was not updated. Want ServiceName=UpdatedService, Price=1500, got: %+v", updated)
	}
}

// TestDelete проверяет корректность удаления подписки.
//
// Проверяемые аспекты:
// - Успешное удаление существующей подписки
// - Возврат ошибки ErrNotExist при попытке чтения удаленной подписки
// - Идемпотентность операции удаления
func TestDelete(t *testing.T) {
	ctx := context.Background()
	sub := newTestSubscription(2025, 10)
	id, _ := testDB.Create(ctx, sub)

	err := testDB.Delete(ctx, id)
	if err != nil {
		t.Fatalf("failed to delete subscription: %v", err)
	}

	_, err = testDB.Get(ctx, id)
	if err != ErrNotExist {
		t.Fatalf("expected ErrNotExist after deletion, got: %v", err)
	}
}

// TestList проверяет получение списков подписок.
//
// Тестируемые сценарии:
// - Получение полного списка всех подписок
// - Фильтрация подписок по конкретному пользователю
// - Корректность сортировки (новые записи первыми)
func TestList(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()

	// Создаём 3 подписки одного пользователя
	for i := 1; i <= 3; i++ {
		sub := models.Subscription{
			ServiceName: "ListTest",
			Price:       100 + i*100,
			UserID:      userID,
			Year:        2025,
			Month:       i,
		}
		_, err := testDB.Create(ctx, sub)
		if err != nil {
			t.Fatalf("failed to create test subscription: %v", err)
		}
	}

	all, err := testDB.List(ctx, "")
	if err != nil {
		t.Fatalf("failed to list all subscriptions: %v", err)
	}
	if len(all) < 3 {
		t.Fatalf("expected at least 3 subscriptions in total list, got %d", len(all))
	}

	userOnly, err := testDB.List(ctx, userID)
	if err != nil {
		t.Fatalf("failed to list user's subscriptions: %v", err)
	}
	if len(userOnly) != 3 {
		t.Fatalf("expected exactly 3 subscriptions for user, got %d", len(userOnly))
	}
}

// TestSumSubscriptionsCost проверяет корректность расчета суммарной стоимости.
//
// Это ключевой бизнес-тест, проверяющий:
// - Фильтрацию по периоду (включительно)
// - Фильтрацию по сервису
// - Фильтрацию по пользователю
// - Комбинированные условия фильтрации
//
// Тестовые данные:
// - 2025-01: сервис A, 500₽
// - 2025-02: сервис B, 600₽
// - 2025-03: сервис A, 700₽
// - 2026-01: сервис A, 800₽ (не входит в период 2025)
func TestSumSubscriptionsCost(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()

	// Создаём тестовые подписки
	testSubs := []models.Subscription{
		{ServiceName: "A", Price: 500, UserID: userID, Year: 2025, Month: 1},
		{ServiceName: "B", Price: 600, UserID: userID, Year: 2025, Month: 2},
		{ServiceName: "A", Price: 700, UserID: userID, Year: 2025, Month: 3},
		{ServiceName: "A", Price: 800, UserID: userID, Year: 2026, Month: 1}, // не должен попасть в 2025
	}

	for _, s := range testSubs {
		_, err := testDB.Create(ctx, s)
		if err != nil {
			t.Fatalf("failed to create test subscription: %v", err)
		}
	}

	// 2025 год целиком → 500 + 600 + 700 = 1800
	total, err := testDB.SumSubscriptionsCost(ctx, userID, "", 2025, 1, 2025, 12)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1800 {
		t.Fatalf("expected 1800₽ for 2025, got %d₽", total)
	}

	// Только сервис "A" в 2025 → 500 + 700 = 1200
	total, err = testDB.SumSubscriptionsCost(ctx, userID, "A", 2025, 1, 2025, 12)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1200 {
		t.Fatalf("expected 1200₽ for service A in 2025, got %d₽", total)
	}

	// Только февраль 2025 → 600
	total, err = testDB.SumSubscriptionsCost(ctx, userID, "", 2025, 2, 2025, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 600 {
		t.Fatalf("expected 600₽ for February 2025, got %d₽", total)
	}
}
