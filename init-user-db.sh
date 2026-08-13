#!/bin/bash
set -e

# Создаём роль orderuser с паролем, если не существует
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'orderuser') THEN
            CREATE ROLE orderuser LOGIN PASSWORD '${ORDERUSER_PASSWORD}';
        END IF;
    END
    \$\$;
    GRANT ALL PRIVILEGES ON DATABASE orders TO orderuser;
EOSQL