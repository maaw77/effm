package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maaw77/effm/config"
	"github.com/maaw77/effm/internal/models"
)

var testDB *SubscriptionDatabase

// TestMain создаёт подключение к базе перед тестами и закрывает его после
func TestMain(m *testing.M) {
	ctx := context.Background()
	connStr := config.InitConnString("config/config.yaml")

	var err error
	testDB, err = NewSubscriptionDatabase(ctx, connStr)
	if err != nil {
		panic(err)
	}

	// Запуск тестов и сохранение кода завершения
	exitCode := m.Run()

	// Закрытие соединения после тестов
	testDB.Close()

	// Завершение процесса с правильным кодом
	os.Exit(exitCode)
}

// helper: создаёт тестовую подписку с указанными датами
func newTestSubscriptionWithDates(start, end *time.Time) models.Subscription {
	sub := models.Subscription{
		ServiceName: "TestService",
		Price:       500,
		UserID:      uuid.New().String(),
	}
	if start != nil {
		sub.StartDate = *start
	} else {
		sub.StartDate = time.Now()
	}
	sub.EndDate = end
	return sub
}

// TestCreateIfNotExist проверяет создание новой подписки и ErrExist при дублировании
func TestCreateIfNotExist(t *testing.T) {
	ctx := context.Background()
	sub := newTestSubscriptionWithDates(nil, nil)

	id, err := testDB.CreateIfNotExist(ctx, sub)
	if err != nil {
		t.Fatalf("CreateIfNotExist failed: %v", err)
	}
	// Удалим подписку после теста
	defer testDB.DeleteSubscription(ctx, id)

	// Повторная попытка должна вернуть ErrExist
	_, err = testDB.CreateIfNotExist(ctx, sub)
	if err != ErrExist {
		t.Fatalf("Ожидали ErrExist, получили %v", err)
	}

	// Проверяем, что подписка создана корректно
	got, err := testDB.GetSubscription(ctx, id)
	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}
	if got.ServiceName != sub.ServiceName || got.UserID != sub.UserID {
		t.Fatalf("Полученные данные не совпадают с ожидаемыми")
	}
}

// TestUpdateSubscription проверяет обновление подписки и ErrNotExist
func TestUpdateSubscription(t *testing.T) {
	ctx := context.Background()
	sub := newTestSubscriptionWithDates(nil, nil)

	id, err := testDB.CreateIfNotExist(ctx, sub)
	if err != nil {
		t.Fatalf("CreateIfNotExist failed: %v", err)
	}
	defer testDB.DeleteSubscription(ctx, id) // откат после теста

	// Обновляем подписку
	sub.ServiceName = "UpdatedService"
	err = testDB.UpdateSubscription(ctx, id, sub)
	if err != nil {
		t.Fatalf("UpdateSubscription failed: %v", err)
	}

	// Проверяем обновление
	updated, _ := testDB.GetSubscription(ctx, id)
	if updated.ServiceName != "UpdatedService" {
		t.Fatalf("Подписка не обновилась")
	}

	// Попытка обновить несуществующую подписку
	err = testDB.UpdateSubscription(ctx, uuid.New().String(), sub)
	if err != ErrNotExist {
		t.Fatalf("Ожидали ErrNotExist, получили %v", err)
	}
}

// TestDeleteSubscription проверяет удаление подписки и ErrNotExist
func TestDeleteSubscription(t *testing.T) {
	ctx := context.Background()
	sub := newTestSubscriptionWithDates(nil, nil)

	id, err := testDB.CreateIfNotExist(ctx, sub)
	if err != nil {
		t.Fatalf("CreateIfNotExist failed: %v", err)
	}

	// Удаляем подписку
	err = testDB.DeleteSubscription(ctx, id)
	if err != nil {
		t.Fatalf("DeleteSubscription failed: %v", err)
	}

	// Попытка удалить снова
	err = testDB.DeleteSubscription(ctx, id)
	if err != ErrNotExist {
		t.Fatalf("Ожидали ErrNotExist, получили %v", err)
	}
}

// TestListSubscriptions проверяет фильтрацию по userID и общий список
func TestListSubscriptions(t *testing.T) {
	ctx := context.Background()

	sub1 := newTestSubscriptionWithDates(nil, nil)
	sub2 := newTestSubscriptionWithDates(nil, nil)

	id1, _ := testDB.CreateIfNotExist(ctx, sub1)
	defer testDB.DeleteSubscription(ctx, id1)
	id2, _ := testDB.CreateIfNotExist(ctx, sub2)
	defer testDB.DeleteSubscription(ctx, id2)

	all, err := testDB.ListSubscriptions(ctx, "")
	if err != nil {
		t.Fatalf("ListSubscriptions failed: %v", err)
	}
	if len(all) < 2 {
		t.Fatalf("Ожидали минимум 2 подписки, получили %d", len(all))
	}

	// Фильтр по пользователю
	listUser1, _ := testDB.ListSubscriptions(ctx, sub1.UserID)
	if len(listUser1) != 1 {
		t.Fatalf("Ожидали 1 подписку для user1, получили %d", len(listUser1))
	}

	listUser2, _ := testDB.ListSubscriptions(ctx, sub2.UserID)
	if len(listUser2) != 1 {
		t.Fatalf("Ожидали 1 подписку для user2, получили %d", len(listUser2))
	}
}

// TestSumSubscriptionsCost проверяет подсчёт суммарной стоимости подписок
func TestSumSubscriptionsCost(t *testing.T) {
	ctx := context.Background()

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	sub1 := newTestSubscriptionWithDates(&start, &end)

	start2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	end2 := start2.AddDate(0, 1, 0)
	sub2 := newTestSubscriptionWithDates(&start2, &end2)

	id1, _ := testDB.CreateIfNotExist(ctx, sub1)
	defer testDB.DeleteSubscription(ctx, id1)
	id2, _ := testDB.CreateIfNotExist(ctx, sub2)
	defer testDB.DeleteSubscription(ctx, id2)

	// Сумма за весь год
	sum, err := testDB.SumSubscriptionsCost(ctx, "", "", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("SumSubscriptionsCost failed: %v", err)
	}
	expected := sub1.Price + sub2.Price
	if sum != expected {
		t.Fatalf("Ожидали сумму %d, получили %d", expected, sum)
	}

	// Сумма за период до июня
	sum, _ = testDB.SumSubscriptionsCost(ctx, "", "", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 5, 31, 0, 0, 0, 0, time.UTC))
	if sum != sub1.Price {
		t.Fatalf("Ожидали сумму %d, получили %d", sub1.Price, sum)
	}

	// Сумма по конкретному пользователю
	sum, _ = testDB.SumSubscriptionsCost(ctx, sub1.UserID, "", start, end)
	if sum != sub1.Price {
		t.Fatalf("Ожидали сумму %d, получили %d", sub1.Price, sum)
	}
}
