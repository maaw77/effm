// internal/server/server_test.go
// Пакет server содержит интеграционные HTTP-тесты для REST API.
// Тесты проверяют все ручки: создание, получение, обновление, удаление, список и подсчёт стоимости.
// Используется в-memory база данных (настоящий PostgreSQL в тестовой БД).

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maaw77/effm/config"
	"github.com/maaw77/effm/internal/database"
	"github.com/maaw77/effm/internal/dto"
	"github.com/maaw77/effm/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServer создаёт сервер с подключением к тестовой БД и очищает таблицы
func setupTestServer(t *testing.T) (*Server, *database.SubscriptionDatabase) {
	connStr := config.InitConnString("config/config.yaml")

	db, err := database.NewSubscriptionDatabase(context.Background(), connStr)
	require.NoError(t, err, "не удалось подключиться к тестовой БД")

	// ← ИСПРАВЛЕНО: RESTART, а не RESTAR
	_, err = db.DBpool.Exec(context.Background(), "TRUNCATE TABLE subscriptions RESTART IDENTITY")
	require.NoError(t, err, "не удалось очистить таблицу")

	srv := NewServer(db)
	gin.SetMode(gin.TestMode)
	return srv, db
}

// TestCreateSubscription_OK проверяет успешное создание подписки
func TestCreateSubscription_OK(t *testing.T) {
	srv, _ := setupTestServer(t)

	reqBody := dto.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus",
		Price:       499,
		UserID:      uuid.New().String(),
		YearMonth:   "2025-07",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	srv.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Правильная и надёжная проверка ответа
	var resp struct {
		ID string `json:"id"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ID)
	assert.Len(t, resp.ID, 36, "ID должен быть валидным UUID (36 символов)")
}

// TestCreateSubscription_Conflict проверяет 409 при дубле
func TestCreateSubscription_Conflict(t *testing.T) {
	srv, db := setupTestServer(t)

	userID := uuid.New().String()
	sub := models.Subscription{
		ServiceName: "Netflix",
		Price:       999,
		UserID:      userID,
		Year:        2025,
		Month:       8,
	}
	_, _ = db.Create(context.Background(), sub) // уже есть

	reqBody := dto.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       999,
		UserID:      userID,
		YearMonth:   "2025-08",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	srv.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "уже существует")
}

// TestGetSubscription_OK проверяет получение по ID
func TestGetSubscription_OK(t *testing.T) {
	srv, db := setupTestServer(t)

	sub := models.Subscription{
		ServiceName: "Spotify",
		Price:       169,
		UserID:      uuid.New().String(),
		Year:        2025,
		Month:       9,
	}
	id, _ := db.Create(context.Background(), sub)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/subscriptions/"+id, nil)
	srv.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Spotify")
	assert.Contains(t, w.Body.String(), "2025-09")
}

// TestUpdateSubscription_OK проверяет обновление
func TestUpdateSubscription_OK(t *testing.T) {
	srv, db := setupTestServer(t)

	sub := models.Subscription{
		ServiceName: "YouTube Premium",
		Price:       249,
		UserID:      uuid.New().String(),
		Year:        2025,
		Month:       10,
	}
	id, _ := db.Create(context.Background(), sub)

	update := dto.UpdateSubscriptionRequest{
		Price:       new(int),
		ServiceName: new(string),
	}
	*update.Price = 299
	*update.ServiceName = "YouTube Premium Family"

	body, _ := json.Marshal(update)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/subscriptions/"+id, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	srv.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что цена изменилась
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/subscriptions/"+id, nil)
	srv.Router.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), "299")
	assert.Contains(t, w.Body.String(), "Family")
}

// TestDeleteSubscription_OK проверяет удаление
func TestDeleteSubscription_OK(t *testing.T) {
	srv, db := setupTestServer(t)

	sub := models.Subscription{
		ServiceName: "Apple Music",
		Price:       169,
		UserID:      uuid.New().String(),
		Year:        2025,
		Month:       11,
	}
	id, _ := db.Create(context.Background(), sub)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/subscriptions/"+id, nil)
	srv.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Должна быть 404
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/subscriptions/"+id, nil)
	srv.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestListSubscriptions проверяет список
func TestListSubscriptions(t *testing.T) {
	srv, db := setupTestServer(t)

	userID := uuid.New().String()
	for i := 1; i <= 3; i++ {
		sub := models.Subscription{
			ServiceName: "TestService",
			Price:       100 + i*100,
			UserID:      userID,
			Year:        2025,
			Month:       i,
		}
		db.Create(context.Background(), sub)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/subscriptions?user_id="+userID, nil)
	srv.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []dto.SubscriptionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp, 3)
}

// TestSumSubscriptionsCost — самый важный тест!
func TestSumSubscriptionsCost(t *testing.T) {
	srv, db := setupTestServer(t)

	userID := uuid.New().String()
	subs := []models.Subscription{
		{ServiceName: "A", Price: 500, UserID: userID, Year: 2025, Month: 1},
		{ServiceName: "B", Price: 600, UserID: userID, Year: 2025, Month: 2},
		{ServiceName: "A", Price: 700, UserID: userID, Year: 2025, Month: 3},
	}
	for _, s := range subs {
		db.Create(context.Background(), s)
	}

	tests := []struct {
		name     string
		start    string
		end      string
		expected int
	}{
		{"весь 2025", "2025-01", "2025-12", 1800},
		{"только сервис A", "2025-01", "2025-12", 1200},
		{"февраль 2025", "2025-02", "2025-02", 600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/subscriptions/total?start=" + tt.start + "&end=" + tt.end
			if tt.name == "только сервис A" {
				url += "&service=A"
			}

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", url, nil)
			srv.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var resp dto.TotalCostResponse
			json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Equal(t, tt.expected, resp.Total)
		})
	}
}
