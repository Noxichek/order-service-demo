#!/bin/bash
set -e

echo "==> Останавливаем локальный PostgreSQL (если есть)..."
sudo service postgresql stop 2>/dev/null || true
sudo systemctl stop postgresql 2>/dev/null || true

echo "==> Запускаем Docker-контейнеры..."
sudo docker compose up -d

echo "==> Ожидание готовности PostgreSQL..."
until sudo docker exec order-postgres pg_isready -U orderuser -d orders >/dev/null 2>&1; do
    sleep 1
done

echo "==> Выдаём права на схему public..."
sudo docker exec -i order-postgres psql -U postgres -d orders \
    -c "GRANT ALL ON SCHEMA public TO orderuser;" || {
        echo "❌ Не удалось выдать права. Проверьте роль postgres."
        exit 1
    }

echo "==> Применяем миграции..."
sudo docker exec -i order-postgres psql -U orderuser -d orders < backend/migrations/001_init.sql

echo "==> Создаём Kafka-топик (если отсутствует)..."
sudo docker exec order-kafka kafka-topics --create \
    --topic orders \
    --bootstrap-server localhost:9092 \
    --partitions 1 \
    --replication-factor 1 || true

# Проверка зависимостей фронтенда
if [ ! -d "frontend/node_modules" ]; then
    echo "❌ Не найдены node_modules. Выполните 'cd frontend && npm install'"
    exit 1
fi

echo "==> Запускаем бэкенд и фронтенд..."
trap 'kill 0' SIGINT SIGTERM EXIT
(cd backend && go run ./cmd/server) &
(cd frontend && npm start) &
wait
