// internal/database/database.go
// Пакет database реализует слой доступа к данным (repository) для работы с подписками пользователей.
// Используется PostgreSQL + pgxpool. Все операции асинхронно-безопасны и работают через context.
//
// Основные принципы:
//   - Одна запись в таблице = оплата за конкретный месяц (модель "monthly billing")
//   - Уникальность: один пользователь — один сервис — один месяц
//   - Все ошибки логируются, критичные бизнес-ошибки возвращаются наружу
package database

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maaw77/effm/internal/models"
)

// Предопределённые ошибки для бизнес-логики
var (
	ErrNotExist = errors.New("подписка не найдена")
	ErrConflict = errors.New("подписка на этот сервис за указанный месяц уже существует")
)

// SubscriptionDatabase — основной объект для работы с БД
type SubscriptionDatabase struct {
	DBpool *pgxpool.Pool
}

// NewSubscriptionDatabase создаёт пул соединений с PostgreSQL.
// Возвращает готовый к работе объект или ошибку подключения.
func NewSubscriptionDatabase(ctx context.Context, connStr string) (*SubscriptionDatabase, error) {
	if connStr == "" {
		return nil, errors.New("строка подключения пустая")
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("невозможно распарсить connString: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать пул соединений: %w", err)
	}

	log.Printf("успешно подключено к PostgreSQL | pool_max_conns=%d", config.MaxConns)
	return &SubscriptionDatabase{DBpool: pool}, nil
}

// Close закрывает пул соединений с БД
func (db *SubscriptionDatabase) Close() {
	if db.DBpool != nil {
		db.DBpool.Close()
		log.Println("соединение с PostgreSQL закрыто")
	}
}

// Create создаёт запись о подписке за конкретный месяц.
// Возвращает UUID созданной записи или ошибку (в т.ч. ErrConflict при дублировании).
func (db *SubscriptionDatabase) Create(ctx context.Context, sub models.Subscription) (string, error) {
	var id string
	err := db.DBpool.QueryRow(ctx,
		`INSERT INTO subscriptions (service_name, price, user_id, year, month)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		sub.ServiceName, sub.Price, sub.UserID, sub.Year, sub.Month,
	).Scan(&id)

	if err != nil {
		if isUniqueViolation(err) {
			log.Printf("[ПРЕДУПРЕЖДЕНИЕ] конфликт: подписка уже существует | пользователь=%s сервис=%s %d-%02d",
				sub.UserID, sub.ServiceName, sub.Year, sub.Month)
			return "", ErrConflict
		}
		log.Printf("ошибка создания подписки | ошибка=%v", err)
		return "", err
	}

	log.Printf("подписка успешно создана | id=%s пользователь=%s сервис=%s год=%d месяц=%02d цена=%d₽",
		id, sub.UserID, sub.ServiceName, sub.Year, sub.Month, sub.Price)

	return id, nil
}

// Get возвращает подписку по её UUID
func (db *SubscriptionDatabase) Get(ctx context.Context, id string) (models.Subscription, error) {
	var sub models.Subscription
	err := db.DBpool.QueryRow(ctx,
		`SELECT id, service_name, price, user_id, year, month, created_at, updated_at
		 FROM subscriptions WHERE id = $1`,
		id,
	).Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID,
		&sub.Year, &sub.Month, &sub.CreatedAt, &sub.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return sub, ErrNotExist
	}
	if err != nil {
		log.Printf("ошибка получения подписки id=%s | ошибка=%v", id, err)
		return sub, err
	}
	return sub, nil
}

// Update изменяет название сервиса и/или цену существующей подписки.
// Месяц и год изменить нельзя — это историческая запись.
func (db *SubscriptionDatabase) Update(ctx context.Context, id string, updates models.Subscription) error {
	cmd, err := db.DBpool.Exec(ctx,
		`UPDATE subscriptions
		 SET service_name = COALESCE(NULLIF($1, ''), service_name),
		     price = COALESCE($2, price),
		     updated_at = NOW()
		 WHERE id = $3`,
		updates.ServiceName, nilIfZero(updates.Price), id,
	)
	if err != nil {
		log.Printf("ошибка обновления подписки id=%s | ошибка=%v", id, err)
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotExist
	}
	log.Printf("подписка успешно обновлена | id=%s", id)
	return nil
}

// Delete удаляет подписку по UUID
func (db *SubscriptionDatabase) Delete(ctx context.Context, id string) error {
	cmd, err := db.DBpool.Exec(ctx, `DELETE FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		log.Printf("ошибка удаления подписки id=%s | ошибка=%v", id, err)
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotExist
	}
	log.Printf("подписка успешно удалена | id=%s", id)
	return nil
}

// List возвращает все подписки (или только одного пользователя), отсортированные по дате (новые сверху)
func (db *SubscriptionDatabase) List(ctx context.Context, userID string) ([]models.Subscription, error) {
	query := `SELECT id, service_name, price, user_id, year, month, created_at, updated_at
	          FROM subscriptions`
	var args []interface{}
	if userID != "" {
		query += " WHERE user_id = $1"
		args = append(args, userID)
	}
	query += " ORDER BY year DESC, month DESC"

	rows, err := db.DBpool.Query(ctx, query, args...)
	if err != nil {
		log.Printf("ошибка получения списка подписок | ошибка=%v", err)
		return nil, err
	}
	defer rows.Close()

	var result []models.Subscription
	for rows.Next() {
		var sub models.Subscription
		if err := rows.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID,
			&sub.Year, &sub.Month, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, sub)
	}
	return result, rows.Err()
}

// SumSubscriptionsCost — главный бизнес-метод: считает суммарную стоимость подписок за период.
// Период задаётся включительно (например, 2025-01 по 2025-12 = весь 2025 год).
func (db *SubscriptionDatabase) SumSubscriptionsCost(
	ctx context.Context,
	userID, serviceName string,
	startYear, startMonth, endYear, endMonth int,
) (int, error) {

	query := `SELECT COALESCE(SUM(price), 0)
	          FROM subscriptions
	          WHERE (year > $1 OR (year = $1 AND month >= $2))
	            AND (year < $3 OR (year = $3 AND month <= $4))`

	args := []interface{}{startYear, startMonth, endYear, endMonth}
	argIdx := 5

	if userID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if serviceName != "" {
		query += " AND service_name = $" + fmt.Sprint(argIdx)
		args = append(args, serviceName)
	}

	var total int
	err := db.DBpool.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		log.Printf("ошибка подсчёта общей стоимости | ошибка=%v", err)
		return 0, err
	}

	// Самый важный лог в приложении — всегда видно, что и сколько посчитали
	log.Printf("подсчитана общая стоимость подписок | пользователь=%s сервис=%s период=%d-%02d..%d-%02d итог=%d₽",
		userID, serviceName, startYear, startMonth, endYear, endMonth, total)

	return total, nil
}

// ——————————————————————————————————————————————————————————————————————
// Вспомогательные функции

// isUniqueViolation проверяет, что ошибка — это нарушение уникального индекса
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return errMsg == `ERROR: duplicate key value violates unique constraint "subscriptions_user_id_service_name_year_month_key" (SQLSTATE 23505)`
}

// nilIfZero помогает передать NULL в COALESCE при обновлении цены
func nilIfZero(v int) interface{} {
	if v == 0 {
		return nil
	}
	return v
}
