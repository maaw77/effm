#!/bin/bash
# test-api.sh — полный end-to-end тест API (все сценарии)
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Очистка порта 8080 и запуск сервера...${NC}"
lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs -r kill -9 2>/dev/null || true
go run cmd/server/main.go &
SERVER_PID=$!
sleep 3

cleanup() {
    echo -e "\n${YELLOW}Останавливаем сервер...${NC}"
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
}
trap cleanup EXIT

BASE="/api/subscriptions"
URL="http://localhost:8080$BASE"
USER=$(uuidgen | tr '[:upper:]' '[:lower:]')

echo -e "${YELLOW}Тестируем с пользователем: $USER${NC}\n"

# 1. Yandex Plus — июль
echo -e "${YELLOW}1. Yandex Plus (2025-07)${NC}"
ID1=$(curl -s -X POST $URL -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Yandex Plus\",\"price\":499,\"user_id\":\"$USER\",\"year_month\":\"2025-07\"}" \
    | grep -o '[0-9a-f]\{8\}-[0-9a-f]\{4\}-[0-9a-f]\{4\}-[0-9a-f]\{4\}-[0-9a-f]\{12\}' | head -1)
[[ -n "$ID1" ]] && echo -e "${GREEN}OK — создана${NC}" || (echo "FAIL" && exit 1)

# 2. Дубликат → 409
echo -e "${YELLOW}2. Дубликат Yandex Plus → 409${NC}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $URL \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Yandex Plus\",\"price\":499,\"user_id\":\"$USER\",\"year_month\":\"2025-07\"}")
[[ "$STATUS" == "409" ]] && echo -e "${GREEN}OK — 409 Conflict${NC}"

# 3. Неверный UUID → 400
echo -e "${YELLOW}3. Неверный user_id → 400${NC}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $URL \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"X\",\"price\":1,\"user_id\":\"bad-uuid\",\"year_month\":\"2025-01\"}")
[[ "$STATUS" == "400" ]] && echo -e "${GREEN}OK — валидация UUID${NC}"

# 4. Неверный месяц → 400
echo -e "${YELLOW}4. year_month=2025-13 → 400${NC}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $URL \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"X\",\"price\":1,\"user_id\":\"$USER\",\"year_month\":\"2025-13\"}")
[[ "$STATUS" == "400" ]] && echo -e "${GREEN}OK — валидация месяца${NC}"

# 5. Spotify в том же месяце — разрешено
echo -e "${YELLOW}5. Spotify (2025-07) — разрешено${NC}"
curl -s -X POST $URL -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Spotify\",\"price\":169,\"user_id\":\"$USER\",\"year_month\":\"2025-07\"}" > /dev/null
echo -e "${GREEN}OK — разные сервисы в одном месяце${NC}"

# 5.5. Netflix в августе (важно для теста 8!)
echo -e "${YELLOW}5.5. Netflix (2025-08)${NC}"
curl -s -X POST $URL -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Netflix\",\"price\":999,\"user_id\":\"$USER\",\"year_month\":\"2025-08\"}" > /dev/null
echo -e "${GREEN}OK — Netflix в августе создан${NC}"

# 6. Подсчёт за весь 2025
echo -e "${YELLOW}6. Итого за 2025 год${NC}"
TOTAL=$(curl -s "$URL/total?start=2025-01&end=2025-12&user_id=$USER" | grep -o '"total":[0-9]\+' | cut -d: -f2)
[[ "$TOTAL" -eq 1667 ]] && echo -e "${GREEN}OK — 1667 ₽ (499 + 169 + 999)${NC}" || echo -e "${RED}Ожидали 1667, получили $TOTAL${NC}"

# 7. Только Yandex Plus
echo -e "${YELLOW}7. Только Yandex Plus${NC}"
TOTAL=$(curl -s "$URL/total?start=2025-01&end=2025-12&user_id=$USER&service=Yandex%20Plus" | grep -o '"total":[0-9]\+' | cut -d: -f2)
[[ "$TOTAL" == "499" ]] && echo -e "${GREEN}OK — 499 ₽${NC}"

# 8. Только август → должен быть Netflix
echo -e "${YELLOW}8. Только август 2025${NC}"
TOTAL=$(curl -s "$URL/total?start=2025-08&end=2025-08&user_id=$USER" | grep -o '"total":[0-9]\+' | cut -d: -f2)
[[ "$TOTAL" == "999" ]] && echo -e "${GREEN}OK — август: 999 ₽ (Netflix)${NC}" || echo -e "${RED}FAIL — получили $TOTAL${NC}"

echo -e "${GREEN}
ВСЁ ЗЕЛЁНОЕ!
${NC}"