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

# Функция для проверки статуса
check_status() {
    local expected=$1
    local actual=$2
    local test_name=$3
    
    if [[ "$actual" -eq "$expected" ]]; then
        echo -e "${GREEN}OK — $test_name${NC}"
        return 0
    else
        echo -e "${RED}FAIL — $test_name (ожидался $expected, получен $actual)${NC}"
        return 1
    fi
}

# 1. Yandex Plus — июль 2025
echo -e "${YELLOW}1. Создаём Yandex Plus (2025-07)${NC}"
ID1=$(curl -s -X POST "$URL" \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Yandex Plus\",\"price\":499,\"user_id\":\"$USER\",\"year_month\":\"2025-07\"}" \
    | grep -Eo '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' || echo "")

if [[ -n "$ID1" ]]; then
    echo -e "${GREEN}OK — создана (id=$ID1)${NC}"
else
    echo -e "${RED}FAIL — не создалась${NC}"
    exit 1
fi

# 2. Дубликат → 409
echo -e "${YELLOW}2. Дубликат Yandex Plus → 409${NC}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$URL" \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Yandex Plus\",\"price\":499,\"user_id\":\"$USER\",\"year_month\":\"2025-07\"}")
check_status 409 "$STATUS" "конфликт дубликатов"

# 3. Неверный UUID → 400
echo -e "${YELLOW}3. Неверный user_id → 400${NC}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$URL" \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"X\",\"price\":1,\"user_id\":\"bad-uuid\",\"year_month\":\"2025-01\"}")
check_status 400 "$STATUS" "валидация UUID"

# 4. Неверный месяц → 400
echo -e "${YELLOW}4. year_month=2025-13 → 400${NC}"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$URL" \
    -H "Content-Type: application/json" \
    -d "{\"service_name\":\"X\",\"price\":1,\"user_id\":\"$USER\",\"year_month\":\"2025-13\"}")
check_status 400 "$STATUS" "валидация месяца"

# 5. Spotify в том же месяце
echo -e "${YELLOW}5. Spotify (2025-07)${NC}"
ID2=$(curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Spotify\",\"price\":169,\"user_id\":\"$USER\",\"year_month\":\"2025-07\"}" \
    | grep -Eo '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' || echo "")

if [[ -n "$ID2" ]]; then
    echo -e "${GREEN}OK — создана (id=$ID2)${NC}"
else
    echo -e "${RED}FAIL — не создалась${NC}"
    exit 1
fi

# 6. Netflix в августе
echo -e "${YELLOW}6. Netflix (2025-08)${NC}"
ID3=$(curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"service_name\":\"Netflix\",\"price\":999,\"user_id\":\"$USER\",\"year_month\":\"2025-08\"}" \
    | grep -Eo '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' || echo "")

if [[ -n "$ID3" ]]; then
    echo -e "${GREEN}OK — создана (id=$ID3)${NC}"
else
    echo -e "${RED}FAIL — не создалась${NC}"
    exit 1
fi

# 7. Итого за 2025 год
echo -e "${YELLOW}7. Итого за 2025 год${NC}"
TOTAL=$(curl -s "http://localhost:8080/api/subscriptions/total?start=2025-01&end=2025-12&user_id=$USER" \
    | grep -o '"total":[0-9]\+' | cut -d: -f2)

if [[ "$TOTAL" -eq 1667 ]]; then
    echo -e "${GREEN}OK — 1667 ₽ (499 + 169 + 999)${NC}"
else
    echo -e "${RED}FAIL — ожидали 1667, получили $TOTAL${NC}"
    exit 1
fi

# 8. Только Yandex Plus
echo -e "${YELLOW}8. Только Yandex Plus${NC}"
TOTAL=$(curl -s "http://localhost:8080/api/subscriptions/total?start=2025-01&end=2025-12&user_id=$USER&service=Yandex%20Plus" \
    | grep -o '"total":[0-9]\+' | cut -d: -f2)

if [[ "$TOTAL" == "499" ]]; then
    echo -e "${GREEN}OK — 499 ₽${NC}"
else
    echo -e "${RED}FAIL — ожидали 499, получили $TOTAL${NC}"
    exit 1
fi

# 9. Только август
echo -e "${YELLOW}9. Только август 2025${NC}"
TOTAL=$(curl -s "http://localhost:8080/api/subscriptions/total?start=2025-08&end=2025-08&user_id=$USER" \
    | grep -o '"total":[0-9]\+' | cut -d: -f2)

if [[ "$TOTAL" == "999" ]]; then
    echo -e "${GREEN}OK — август: 999 ₽ (Netflix)${NC}"
else
    echo -e "${RED}FAIL — ожидали 999, получили $TOTAL${NC}"
    exit 1
fi

# 10. Проверка чтения созданных подписок
echo -e "${YELLOW}10. Проверка чтения подписок${NC}"
for id in "$ID1" "$ID2" "$ID3"; do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/api/subscriptions/$id")
    if check_status 200 "$STATUS" "чтение подписки $id"; then
        echo -e "  ${GREEN}✓ Подписка $id доступна${NC}"
    fi
done

# Финал
echo -e "${GREEN}
╔══════════════════════════════════════════╗
║   ВСЁ ЗЕЛЁНОЕ! API РАБОТАЕТ ИДЕАЛЬНО! 🔥   ║
╚══════════════════════════════════════════╝
${NC}"
echo -e "${YELLOW}Контейнеры продолжают работать. Данные сохранены в томе.${NC}"