#!/bin/bash
set -e

# Остановка локального PostgreSQL
sudo service postgresql stop 2>/dev/null || true
sudo systemctl stop postgresql 2>/dev/null || true

# Запуск инфраструктуры
sudo docker compose up -d

# Ожидание готовности PostgreSQL
until sudo docker exec order-postgres pg_isready -U orderuser -d orders >/dev/null 2>&1; do
    sleep 1
done

# Выдача прав
sudo docker exec -i order-postgres psql -U postgres -d orders \
    -c "GRANT ALL ON SCHEMA public TO orderuser;" || { echo "❌ Ошибка выдачи прав"; exit 1; }

# Миграции
sudo docker exec -i order-postgres psql -U orderuser -d orders < backend/migrations/001_init.sql

# Топик Kafka
sudo docker exec order-kafka kafka-topics --create \
    --topic orders \
    --bootstrap-server localhost:9092 \
    --partitions 1 \
    --replication-factor 1 || true

# Запуск Go-сервера
cd backend && go run ./cmd/server