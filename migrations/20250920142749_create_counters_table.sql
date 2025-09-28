-- +goose Up
CREATE TABLE IF NOT EXISTS counters (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    value BIGINT NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS counters;