-- +goose Up
-- Создаем таблицу для метрик типа Gauge
CREATE TABLE IF NOT EXISTS gauges (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value DOUBLE PRECISION NOT NULL
);

-- +goose Down
-- Откат: удаляем таблицу gauges
DROP TABLE IF EXISTS gauges;