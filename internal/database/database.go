// Package database реализует работу с таблицей subscriptions в PostgreSQL.
// Использует pgxpool для пуллинга соединений и context для управления таймаутами/отменой.
// Методы поддерживают CRUD операции и подсчет суммарной стоимости подписок.
//
// Все методы принимают context.Context для контроля таймаутов и отмены запросов.
// Методы CreateIfNotExist и UpdateSubscription выполняются в транзакциях.
// Ошибки:
//   - ErrExist    : запись уже существует (для CreateIfNotExist)
//   - ErrNotExist : запись не найдена
package database

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maaw77/effm/internal/models"
)

var (
	ErrNotExist = errors.New("record does not exist")
	ErrExist    = errors.New("record already exists")
)

// SubscriptionDatabase хранит пул соединений с PostgreSQL
type SubscriptionDatabase struct {
	DBpool *pgxpool.Pool
}

// NewSubscriptionDatabase создаёт новое подключение к базе по connection string.
// Возвращает ошибку, если строка подключения пустая или соединение не удалось.
func NewSubscriptionDatabase(ctx context.Context, connStr string) (*SubscriptionDatabase, error) {
	if connStr == "" {
		return nil, errors.New("connection string is empty")
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}

	db := &SubscriptionDatabase{}
	db.DBpool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	log.Println("✅ Connected to PostgreSQL")
	return db, nil
}

// Close закрывает пул соединений с базой данных.
func (db *SubscriptionDatabase) Close() {
	if db.DBpool != nil {
		db.DBpool.Close()
		log.Println("🔒 Connection to PostgreSQL closed")
	}
}

// CreateIfNotExist вставляет новую подписку в базу, если нет подписки
// с тем же user_id, service_name и start_date.
// Возвращает ID подписки или ErrExist, если такая подписка уже есть.
// Операция выполняется в транзакции.
func (db *SubscriptionDatabase) CreateIfNotExist(ctx context.Context, sub models.Subscription) (string, error) {
	var id string

	tx, err := db.DBpool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	// Проверка существующей подписки
	err = tx.QueryRow(ctx,
		`SELECT id FROM subscriptions WHERE user_id=$1 AND service_name=$2 AND start_date=$3`,
		sub.UserID, sub.ServiceName, sub.StartDate).Scan(&id)

	if err == nil {
		return id, ErrExist
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	// Если не найдено, создаём
	err = tx.QueryRow(ctx,
		`INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate).Scan(&id)
	if err != nil {
		return "", err
	}

	if err = tx.Commit(ctx); err != nil {
		return "", err
	}

	log.Printf("✅ Subscription created: %s\n", id)
	return id, nil
}

// UpdateSubscription обновляет подписку по ID.
// Если запись не найдена, возвращает ErrNotExist.
// Операция выполняется в транзакции.
func (db *SubscriptionDatabase) UpdateSubscription(ctx context.Context, id string, sub models.Subscription) error {
	tx, err := db.DBpool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	ct, err := tx.Exec(ctx,
		`UPDATE subscriptions SET service_name=$1, price=$2, start_date=$3, end_date=$4, updated_at=$5
		 WHERE id=$6`,
		sub.ServiceName, sub.Price, sub.StartDate, sub.EndDate, time.Now(), id,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotExist
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	log.Printf("✏️ Subscription updated: %s\n", id)
	return nil
}

// DeleteSubscription удаляет подписку по ID.
// Возвращает ErrNotExist, если запись не найдена.
func (db *SubscriptionDatabase) DeleteSubscription(ctx context.Context, id string) error {
	ct, err := db.DBpool.Exec(ctx, `DELETE FROM subscriptions WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotExist
	}
	log.Printf("🗑 Subscription deleted: %s\n", id)
	return nil
}

// GetSubscription возвращает подписку по ID.
// Возвращает ErrNotExist, если запись не найдена.
func (db *SubscriptionDatabase) GetSubscription(ctx context.Context, id string) (models.Subscription, error) {
	var sub models.Subscription
	err := db.DBpool.QueryRow(
		ctx,
		`SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		 FROM subscriptions WHERE id=$1`,
		id,
	).Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartDate, &sub.EndDate, &sub.CreatedAt, &sub.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return sub, ErrNotExist
	}

	return sub, err
}

// ListSubscriptions возвращает все подписки пользователя, если userID указан,
// или все подписки, если userID пустой.
func (db *SubscriptionDatabase) ListSubscriptions(ctx context.Context, userID string) ([]models.Subscription, error) {
	query := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at FROM subscriptions`
	var args []interface{}
	if userID != "" {
		query += " WHERE user_id=$1"
		args = append(args, userID)
	}

	rows, err := db.DBpool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		var sub models.Subscription
		if err := rows.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartDate, &sub.EndDate, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// SumSubscriptionsCost считает суммарную стоимость подписок за период start..end,
// с опциональной фильтрацией по userID и serviceName.
// Возвращает 0, если подписок не найдено.
func (db *SubscriptionDatabase) SumSubscriptionsCost(ctx context.Context, userID, serviceName string, start, end time.Time) (int, error) {
	query := `SELECT COALESCE(SUM(price),0) FROM subscriptions WHERE start_date >= $1 AND start_date <= $2`
	args := []interface{}{start, end}

	if userID != "" {
		query += " AND user_id=$3"
		args = append(args, userID)
	}
	if serviceName != "" {
		query += " AND service_name=$4"
		args = append(args, serviceName)
	}

	var sum int
	err := db.DBpool.QueryRow(ctx, query, args...).Scan(&sum)
	if err != nil {
		return 0, err
	}
	log.Printf("💰 SumSubscriptionsCost: %d\n", sum)
	return sum, nil
}
