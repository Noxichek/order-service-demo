#!/bin/bash
set -e  # Останавливать скрипт при любой ошибке

# ============================================
# 1. Остановить локальный PostgreSQL (если запущен),
#    чтобы освободить порт 5432 для Docker-контейнера.
#    Если службы нет, команда просто завершится с ошибкой, поэтому || true.
# ============================================
echo "==> Останавливаем локальный PostgreSQL (если есть)..."
sudo service postgresql stop 2>/dev/null || true
sudo systemctl stop postgresql 2>/dev/null || true

# ============================================
# 2. Запустить инфраструктуру (Kafka, PostgreSQL) через Docker Compose.
#    Если контейнеры уже запущены, этот шаг ничего не сломает.
# ============================================
echo "==> Запускаем Docker-контейнеры (PostgreSQL, ZooKeeper, Kafka)..."
sudo docker compose up -d

# ============================================
# 3. Ждать, пока PostgreSQL станет готовым принимать подключения.
#    Проверяем через pg_isready внутри контейнера.
# ============================================
echo "==> Ожидание готовности PostgreSQL..."
until sudo docker exec order-postgres pg_isready -U orderuser -d orders >/dev/null 2>&1; do
    sleep 1
done

# ============================================
# 4. Выдать права роли orderuser на схему public.
#    Это нужно для PostgreSQL 15+, где по умолчанию запрещено создавать таблицы.
#    Если права уже выданы, команда не навредит.
# ============================================
echo "==> Выдаём права на схему public..."
sudo docker exec -it order-postgres psql -U postgres -d orders \
    -c "GRANT ALL ON SCHEMA public TO orderuser;" || true

# ============================================
# 5. Применить миграции (создать таблицы).
#    Файл backend/migrations/001_init.sql содержит все CREATE TABLE.
# ============================================
echo "==> Применяем миграции..."
sudo docker exec -i order-postgres psql -U orderuser -d orders < backend/migrations/001_init.sql

# ============================================
# 6. Создать Kafka-топик orders, если он ещё не создан.
#    Если топик уже существует, команда выведет предупреждение,
#    но мы подавляем его через || true.
# ============================================
echo "==> Создаём Kafka-топик 'orders' (если отсутствует)..."
sudo docker exec order-kafka kafka-topics --create \
    --topic orders \
    --bootstrap-server localhost:9092 \
    --partitions 1 \
    --replication-factor 1 || true

# ============================================
# 7. Запустить бэкенд (Go) и фронтенд (Angular) параллельно.
#    Ловим Ctrl+C, чтобы корректно остановить оба процесса.
# ============================================
echo "==> Запускаем бэкенд и фронтенд..."
trap 'kill 0' SIGINT SIGTERM EXIT

# Запускаем бэкенд в фоне
(cd backend && go run ./cmd/server) &

# Запускаем фронтенд в фоне
(cd frontend && ng serve) &

# Ждём завершения обоих процессов (например, по Ctrl+C)
wait