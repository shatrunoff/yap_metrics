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

// конфигурация для повторных попыток
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

// проверяет, является ли ошибка ошибкой соединения (для агента)
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// Проверяем стандартные сетевые ошибки
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

	// Проверяем net.Error
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Проверяем DNS ошибки
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Проверяем ошибки HTTP
	switch {
	case errors.Is(err, http.ErrHandlerTimeout):
		return true
	case errors.Is(err, http.ErrServerClosed):
		return true
	case errors.Is(err, http.ErrContentLength):
		return true
	}

	// Проверяем строковые представления
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
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure:
		return true
	default:
		return false
	}
}

// возвращает интервал для конкретной попытки
func getBackoffForAttempt(backoffs []time.Duration, attempt int) time.Duration {
	if attempt < 0 || attempt >= len(backoffs) {
		return backoffs[len(backoffs)-1]
	}
	return backoffs[attempt]
}

// приостанавливает выполнение ф-ии с учетом контекста
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

// универсальный тип для ф-ии с параметрами
type RetryableFunc[T any] func(ctx context.Context, args T) error

// Для функций без контекста
type RetryableFuncNoCtx[T any] func(args T) error

// универсальный вызов повторных попыток
func RetrySendWithArgs[T any](ctx context.Context, funcName string, function RetryableFunc[T], args T) error {
	var lastErr error
	var nameFlag bool  // false - agent, true - postgres
	var checkFlag bool // какое условие используем, для агента и для postgres - разные

	retryConfig := DefaultRetryConfig()
	attempt := 0

	// проверяем какие условия для retry проверять далее
	// так как функция универсальная
	agentNames := []string{"SendBatch"}
	postgresNames := []string{"UpdateGauge", "UpdateCounter", "getGauge", "getCounter", "getAllGauges", "getAllCounters"}

	for _, name := range agentNames {
		if name == funcName {
			nameFlag = false
		}
	}

	for _, name := range postgresNames {
		if name == funcName {
			nameFlag = true
		}
	}

	for attempt < retryConfig.MaxAttempts {
		select {
		case <-ctx.Done():
			log.Printf("%s: func canceled by context", funcName)
			return ctx.Err()
		default:
		}

		// получаем результат исполнения ф-ии
		err := function(ctx, args)

		// если завершилась успешно
		if err == nil {
			if attempt > 0 {
				log.Printf("%s: func successed after %d retries", funcName, attempt)
			}
			return nil
		}

		lastErr = err

		// делаем ветвление логики для проверки типа ошибки для агента и для работы с БД
		if nameFlag {
			if pgErr, ok := err.(*pgconn.PgError); ok {
				checkFlag = isConnectionPostgresError(pgErr)
			}
		} else {
			checkFlag = IsConnectionError(err)
		}

		// если агент - сетевая ошибка И это не последняя попытка
		// если postgres - ошибка транспорта И это не последняя попытка
		if checkFlag && attempt < retryConfig.MaxAttempts {

			attempt++

			// получаем интервал ожидания для текущей попытки
			backoff := getBackoffForAttempt(retryConfig.Backoffs, attempt-1)

			log.Printf("%s: func attempt [%d/%d] failed, retry in %v: %v",
				funcName, attempt, retryConfig.MaxAttempts, backoff, err)

			// ждем следующую попытку
			if err := sleepWithContext(ctx, backoff); err != nil {
				return err
			}
		} else {
			break
		}
	}

	return fmt.Errorf("%s: func failed after %d retries: %w", funcName, retryConfig.MaxAttempts, lastErr)
}

// Упрощенный вызов retry без контекста
func RetrySendWithArgsNoCtx[T any](funcName string, function RetryableFuncNoCtx[T], args T) error {
	// Используем background context
	return RetrySendWithArgs(context.Background(), funcName, func(ctx context.Context, args T) error {
		return function(args)
	}, args)
}
