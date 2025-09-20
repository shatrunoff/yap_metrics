-- +goose Up
-- Создаем таблицу для метрик типа Counter
CREATE TABLE IF NOT EXISTS counters (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value BIGINT NOT NULL DEFAULT 0
);

-- +goose Down
-- Откат: удаляем таблицу counters
DROP TABLE IF EXISTS counters;


goose -dir migrations postgres "postgres://postgres:69folunu@localhost:5432/postgres?sslmode=disable" 
create init sql
