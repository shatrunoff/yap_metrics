-- Создание таблицы для gauge метрик
CREATE TABLE IF NOT EXISTS gauges (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    value DOUBLE PRECISION NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы для counter метрик
CREATE TABLE IF NOT EXISTS counters (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    value BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_gauges_name ON gauges(name);
CREATE INDEX IF NOT EXISTS idx_counters_name ON counters(name);