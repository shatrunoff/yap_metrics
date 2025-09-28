package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// описывает базовые параметры retry
type RetryConfig struct {
	MaxAttempts int
	Backoffs    []time.Duration
}

// возвращает конфигурацию по умолчанию (1s, 3s, 5s)
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		Backoffs:    []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second},
	}
}

// определяет стратегию повторных попыток
type RetryStrategy interface {
	MaxAttempts() int
	Backoff(attempt int) time.Duration
	ShouldRetry(err error) bool
}

// базовая реализация общих частей стратегии
type baseStrategy struct{ cfg RetryConfig }

func (b baseStrategy) MaxAttempts() int { return b.cfg.MaxAttempts }
func (b baseStrategy) Backoff(attempt int) time.Duration {
	return getBackoffForAttempt(b.cfg.Backoffs, attempt)
}

// стратегия для сетевых ошибок (агент)
type NetworkRetryStrategy struct{ baseStrategy }

func NewNetworkRetryStrategy(cfg RetryConfig) RetryStrategy {
	return NetworkRetryStrategy{baseStrategy{cfg: cfg}}
}

func (NetworkRetryStrategy) ShouldRetry(err error) bool { return IsConnectionError(err) }

// стратегия для ошибок подключения к PostgreSQL
type PostgresRetryStrategy struct{ baseStrategy }

func NewPostgresRetryStrategy(cfg RetryConfig) RetryStrategy {
	return PostgresRetryStrategy{baseStrategy{cfg: cfg}}
}

func (PostgresRetryStrategy) ShouldRetry(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return isConnectionPostgresError(pgErr)
	}
	return false
}

// проверяет, является ли ошибка сетевой
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, io.EOF):
		return true
	case errors.Is(err, syscall.ECONNREFUSED):
		return true
	case errors.Is(err, syscall.ECONNRESET):
		return true
	case errors.Is(err, syscall.ETIMEDOUT):
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	switch {
	case errors.Is(err, http.ErrHandlerTimeout):
		return true
	case errors.Is(err, http.ErrServerClosed):
		return true
	case errors.Is(err, http.ErrContentLength):
		return true
	}

	errorStr := err.Error()
	switch {
	case strings.Contains(errorStr, "connection refused"):
		return true
	case strings.Contains(errorStr, "no such host"):
		return true
	case strings.Contains(errorStr, "network is unreachable"):
		return true
	case strings.Contains(errorStr, "timeout"):
		return true
	case strings.Contains(errorStr, "TLS handshake"):
		return true
	case strings.Contains(errorStr, "EOF"):
		return true
	case strings.Contains(errorStr, "broken pipe"):
		return true
	case strings.Contains(errorStr, "reset by peer"):
		return true
	}

	return false
}

// проверяет, является ли ошибка PostgreSQL ошибкой соединения
func isConnectionPostgresError(pgErr *pgconn.PgError) bool {
	if pgErr == nil {
		return false
	}

	if len(pgErr.Code) >= 2 && pgErr.Code[:2] == "08" {
		return true
	}
	switch pgErr.Code {
	// Класс 08 - Ошибки соединения
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure:
		return true

	// Класс 40 - Откат транзакции
	case pgerrcode.TransactionRollback, // 40000
		pgerrcode.SerializationFailure, // 40001
		pgerrcode.DeadlockDetected:     // 40P01
		return true

	// Класс 57 - Ошибка оператора
	case pgerrcode.CannotConnectNow: // 57P03
		return true
	}
	return false
}

// возвращает интервал для конкретной попытки
func getBackoffForAttempt(backoffs []time.Duration, attempt int) time.Duration {
	if attempt < 0 || attempt >= len(backoffs) {
		return backoffs[len(backoffs)-1]
	}
	return backoffs[attempt]
}

// приостанавливает выполнение с учетом контекста
func sleepWithContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// универсальные типы функций для retry
type RetryableFunc[T any] func(ctx context.Context, args T) error
type RetryableFuncNoCtx[T any] func(args T) error

// выполняет функцию с повторными попытками согласно стратегии
func RetryWithStrategy[T any](ctx context.Context, name string, strategy RetryStrategy, function RetryableFunc[T], args T) error {
	var lastErr error
	max := strategy.MaxAttempts()

	for attempt := 0; attempt < max; attempt++ {
		select {
		case <-ctx.Done():
			log.Printf("%s: canceled by context", name)
			return ctx.Err()
		default:
		}

		err := function(ctx, args)
		if err == nil {
			if attempt > 0 {
				log.Printf("%s: succeeded after %d retries", name, attempt)
			}
			return nil
		}

		lastErr = err

		// нет смысла ждать, если следующей попытки не будет
		if attempt+1 >= max || !strategy.ShouldRetry(err) {
			break
		}

		backoff := strategy.Backoff(attempt)
		log.Printf("%s: attempt [%d/%d] failed, retry in %v: %v", name, attempt+1, max, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}

	return fmt.Errorf("%s: failed after %d retries: %w", name, max, lastErr)
}

// RetryNoCtxWithStrategy – версия без контекста
func RetryNoCtxWithStrategy[T any](name string, strategy RetryStrategy, function RetryableFuncNoCtx[T], args T) error {
	return RetryWithStrategy(context.Background(), name, strategy, func(ctx context.Context, a T) error { return function(a) }, args)
}

// Удобные обертки со стратегиями по умолчанию
func RetryPostgres[T any](ctx context.Context, name string, function RetryableFunc[T], args T) error {
	return RetryWithStrategy(ctx, name, NewPostgresRetryStrategy(DefaultRetryConfig()), function, args)
}

func RetryNetworkNoCtx[T any](name string, function RetryableFuncNoCtx[T], args T) error {
	return RetryNoCtxWithStrategy(name, NewNetworkRetryStrategy(DefaultRetryConfig()), function, args)
}
