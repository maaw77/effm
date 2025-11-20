#!/usr/bin/env bash
# test-api.sh — e2e-тесты API на уже запущенном docker-compose стеке

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Проверяем, что сервер действительно живой
echo -e "${YELLOW}Проверка доступности сервера...${NC}"
if ! curl -s http://localhost:8080/swagger/index.html > /dev/null; then
    echo -e "${RED}Сервер недоступен на http://localhost:8080${NC}"
    echo -e "${RED}Запусти: docker compose up -d${NC}"
    exit 1
fi
echo -e "${GREEN}Сервер уже работает! 🚀${NC}\n"

BASE="/api/subscriptions"
URL="http://localhost:8080$BASE"
USER=$(uuidgen | tr '[:upper:]' '[:lower:]')
echo -e "${YELLOW}Тестируем с пользователем: $USER${NC}\n"

# 1. Yandex Plus — июль 2025
echo -e "${YELLOW}1. Создаём Yandex Plus (2025-07)${NC}"
ID1=$(curl -s -X POST "$URL" \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Yandex Plus\",\"price\":499,\"user_id\":\"$USER\",\"year_month\":\"2025-07\"}" \
    | grep -Eo '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' || echo "")

[[ -n "$ID1" ]] && echo -e "${GREEN}OK — создана (id=$ID1)${NC}" || (echo -e "${RED}FAIL — не создалась${NC}" && exit 1)

# 2. Дубликат → 409
echo -e "${YELLOW}2. Дубликат Yandex Plus → 409${NC}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$URL" \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Yandex Plus\",\"price\":499,\"user_id\":\"$USER\",\"year_month\":\"2025-07\"}")
[[ "$STATUS" == "409" ]] && echo -e "${GREEN}OK — 409 Conflict${NC}" || (echo -e "${RED}FAIL — код $STATUS${NC}" && exit 1)

# 3. Неверный UUID → 400
echo -e "${YELLOW}3. Неверный user_id → 400${NC}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$URL" \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"X\",\"price\":1,\"user_id\":\"bad-uuid\",\"year_month\":\"2025-01\"}")
[[ "$STATUS" == "400" ]] && echo -e "${GREEN}OK — валидация UUID${NC}" || (echo -e "${RED}FAIL — код $STATUS${NC}" && exit 1)

# 4. Неверный месяц → 400
echo -e "${YELLOW}4. year_month=2025-13 → 400${NC}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$URL" \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"X\",\"price\":1,\"user_id\":\"$USER\",\"year_month\":\"2025-13\"}")
[[ "$STATUS" == "400" ]] && echo -e "${GREEN}OK — валидация месяца${NC}" || (echo -e "${RED}FAIL — код $STATUS${NC}" && exit 1)

# 5. Spotify в том же месяце
echo -e "${YELLOW}5. Spotify (2025-07)${NC}"
curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Spotify\",\"price\":169,\"user_id\":\"$USER\",\"year_month\":\"2025-07\"}" > /dev/null
echo -e "${GREEN}OK — разные сервисы в одном месяце${NC}"

# 5.5. Netflix в августе
echo -e "${YELLOW}5.5. Netflix (2025-08)${NC}"
curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Netflix\",\"price\":999,\"user_id\":\"$USER\",\"year_month\":\"2025-08\"}" > /dev/null
echo -e "${GREEN}OK — Netflix создан${NC}"

# 6. Итого за 2025 год
echo -e "${YELLOW}6. Итого за 2025 год${NC}"
TOTAL=$(curl -s "http://localhost:8080/api/subscriptions/total?start=2025-01&end=2025-12&user_id=$USER" \
    | grep -o '"total":[0-9]\+' | cut -d: -f2)
[[ "$TOTAL" -eq 1667 ]] && echo -e "${GREEN}OK — 1667 ₽ (499 + 169 + 999)${NC}" \
    || (echo -e "${RED}Ожидали 1667, получили $TOTAL${NC}" && exit 1)

# 7. Только Yandex Plus
echo -e "${YELLOW}7. Только Yandex Plus${NC}"
TOTAL=$(curl -s "http://localhost:8080/api/subscriptions/total?start=2025-01&end=2025-12&user_id=$USER&service=Yandex%20Plus" \
    | grep -o '"total":[0-9]\+' | cut -d: -f2)
[[ "$TOTAL" == "499" ]] && echo -e "${GREEN}OK — 499 ₽${NC}" || (echo -e "${RED}FAIL — получили $TOTAL${NC}" && exit 1)

# 8. Только август
echo -e "${YELLOW}8. Только август 2025${NC}"
TOTAL=$(curl -s "http://localhost:8080/api/subscriptions/total?start=2025-08&end=2025-08&user_id=$USER" \
    | grep -o '"total":[0-9]\+' | cut -d: -f2)
[[ "$TOTAL" == "999" ]] && echo -e "${GREEN}OK — август: 999 ₽ (Netflix)${NC}" || (echo -e "${RED}FAIL — получили $TOTAL${NC}" && exit 1)

# Финал
echo -e "${GREEN}
╔══════════════════════════════════════════╗
║   ВСЁ ЗЕЛЁНОЕ! API РАБОТАЕТ ИДЕАЛЬНО! 🔥   ║
╚══════════════════════════════════════════╝
${NC}"
echo -e "${YELLOW}Контейнеры продолжают работать. Данные сохранены в томе.${NC}"