package utils

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("expected 3 max attempts, got %d", cfg.MaxAttempts)
	}
	if len(cfg.Backoffs) != 3 {
		t.Errorf("expected 3 backoffs, got %d", len(cfg.Backoffs))
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"EOF", io.EOF, true},
		{"ECONNREFUSED", syscall.ECONNREFUSED, true},
		{"ECONNRESET", syscall.ECONNRESET, true},
		{"ETIMEDOUT", syscall.ETIMEDOUT, true},
		{"ErrHandlerTimeout", http.ErrHandlerTimeout, true},
		{"ErrServerClosed", http.ErrServerClosed, true},
		{"other error", errors.New("some error"), false},
		{"connection refused string", errors.New("connection refused"), true},
		{"no such host string", errors.New("no such host"), true},
		{"network unreachable string", errors.New("network is unreachable"), true},
		{"timeout string", errors.New("timeout"), true},
		{"TLS handshake string", errors.New("TLS handshake failed"), true},
		{"EOF string", errors.New("EOF"), true},
		{"broken pipe string", errors.New("broken pipe"), true},
		{"reset by peer string", errors.New("reset by peer"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectionError(tt.err)
			if result != tt.expected {
				t.Errorf("IsConnectionError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsConnectionError_NetError(t *testing.T) {
	err := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("test")}
	if !IsConnectionError(err) {
		t.Error("expected net.OpError to be a connection error")
	}
}

func TestIsConnectionError_DNSError(t *testing.T) {
	err := &net.DNSError{Err: "no such host", Name: "example.com"}
	if !IsConnectionError(err) {
		t.Error("expected net.DNSError to be a connection error")
	}
}

func TestIsConnectionPostgresError(t *testing.T) {
	tests := []struct {
		name     string
		pgErr    *pgconn.PgError
		expected bool
	}{
		{
			name:     "nil error",
			pgErr:    nil,
			expected: false,
		},
		{
			name:     "connection exception",
			pgErr:    &pgconn.PgError{Code: pgerrcode.ConnectionException},
			expected: true,
		},
		{
			name:     "connection does not exist",
			pgErr:    &pgconn.PgError{Code: pgerrcode.ConnectionDoesNotExist},
			expected: true,
		},
		{
			name:     "connection failure",
			pgErr:    &pgconn.PgError{Code: pgerrcode.ConnectionFailure},
			expected: true,
		},
		{
			name:     "serialization failure",
			pgErr:    &pgconn.PgError{Code: pgerrcode.SerializationFailure},
			expected: true,
		},
		{
			name:     "deadlock detected",
			pgErr:    &pgconn.PgError{Code: pgerrcode.DeadlockDetected},
			expected: true,
		},
		{
			name:     "cannot connect now",
			pgErr:    &pgconn.PgError{Code: pgerrcode.CannotConnectNow},
			expected: true,
		},
		{
			name:     "class 08 error",
			pgErr:    &pgconn.PgError{Code: "08001"},
			expected: true,
		},
		{
			name:     "other error",
			pgErr:    &pgconn.PgError{Code: "23505"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConnectionPostgresError(tt.pgErr)
			if result != tt.expected {
				t.Errorf("isConnectionPostgresError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetBackoffForAttempt(t *testing.T) {
	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second}

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"first attempt", 0, 1 * time.Second},
		{"second attempt", 1, 2 * time.Second},
		{"third attempt", 2, 3 * time.Second},
		{"out of range", 5, 3 * time.Second},
		{"negative", -1, 3 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBackoffForAttempt(backoffs, tt.attempt)
			if result != tt.expected {
				t.Errorf("getBackoffForAttempt(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestSleepWithContext(t *testing.T) {
	t.Run("normal sleep", func(t *testing.T) {
		ctx := context.Background()
		err := sleepWithContext(ctx, 10*time.Millisecond)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := sleepWithContext(ctx, 1*time.Second)
		if err == nil {
			t.Error("expected error for canceled context")
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		err := sleepWithContext(ctx, 1*time.Second)
		if err == nil {
			t.Error("expected error for context timeout")
		}
	})
}

func TestNewNetworkRetryStrategy(t *testing.T) {
	cfg := DefaultRetryConfig()
	strategy := NewNetworkRetryStrategy(cfg)

	if strategy.MaxAttempts() != 3 {
		t.Errorf("expected 3 max attempts, got %d", strategy.MaxAttempts())
	}
}

func TestNetworkRetryStrategy_ShouldRetry(t *testing.T) {
	strategy := NewNetworkRetryStrategy(DefaultRetryConfig())

	if !strategy.ShouldRetry(io.EOF) {
		t.Error("expected to retry on EOF")
	}

	if strategy.ShouldRetry(errors.New("some error")) {
		t.Error("expected not to retry on non-connection error")
	}
}

func TestNewPostgresRetryStrategy(t *testing.T) {
	cfg := DefaultRetryConfig()
	strategy := NewPostgresRetryStrategy(cfg)

	if strategy.MaxAttempts() != 3 {
		t.Errorf("expected 3 max attempts, got %d", strategy.MaxAttempts())
	}
}

func TestPostgresRetryStrategy_ShouldRetry(t *testing.T) {
	strategy := NewPostgresRetryStrategy(DefaultRetryConfig())

	pgErr := &pgconn.PgError{Code: pgerrcode.ConnectionException}
	if !strategy.ShouldRetry(pgErr) {
		t.Error("expected to retry on postgres connection error")
	}

	if strategy.ShouldRetry(errors.New("some error")) {
		t.Error("expected not to retry on non-postgres error")
	}
}

func TestRetryWithStrategy_Success(t *testing.T) {
	strategy := NewNetworkRetryStrategy(DefaultRetryConfig())
	called := 0

	fn := func(ctx context.Context, args string) error {
		called++
		return nil
	}

	err := RetryWithStrategy(context.Background(), "test", strategy, fn, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestRetryWithStrategy_SuccessAfterRetry(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 3,
		Backoffs:    []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond},
	}
	strategy := NewNetworkRetryStrategy(cfg)
	called := 0

	fn := func(ctx context.Context, args string) error {
		called++
		if called < 2 {
			return io.EOF
		}
		return nil
	}

	err := RetryWithStrategy(context.Background(), "test", strategy, fn, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called != 2 {
		t.Errorf("expected 2 calls, got %d", called)
	}
}

func TestRetryWithStrategy_FailAfterMaxAttempts(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 3,
		Backoffs:    []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond},
	}
	strategy := NewNetworkRetryStrategy(cfg)

	fn := func(ctx context.Context, args string) error {
		return io.EOF
	}

	err := RetryWithStrategy(context.Background(), "test", strategy, fn, "test")
	if err == nil {
		t.Error("expected error after max attempts")
	}
}

func TestRetryWithStrategy_ContextCanceled(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 3,
		Backoffs:    []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond},
	}
	strategy := NewNetworkRetryStrategy(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fn := func(ctx context.Context, args string) error {
		return io.EOF
	}

	err := RetryWithStrategy(ctx, "test", strategy, fn, "test")
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestRetryNoCtxWithStrategy(t *testing.T) {
	strategy := NewNetworkRetryStrategy(DefaultRetryConfig())
	called := 0

	fn := func(args string) error {
		called++
		return nil
	}

	err := RetryNoCtxWithStrategy("test", strategy, fn, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestRetryPostgres(t *testing.T) {
	called := 0

	fn := func(ctx context.Context, args string) error {
		called++
		return nil
	}

	err := RetryPostgres(context.Background(), "test", fn, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestRetryNetworkNoCtx(t *testing.T) {
	called := 0

	fn := func(args string) error {
		called++
		return nil
	}

	err := RetryNetworkNoCtx("test", fn, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestBaseStrategy_Backoff(t *testing.T) {
	cfg := DefaultRetryConfig()
	strategy := baseStrategy{cfg: cfg}

	backoff := strategy.Backoff(0)
	if backoff != 1*time.Second {
		t.Errorf("expected 1s, got %v", backoff)
	}

	backoff = strategy.Backoff(1)
	if backoff != 3*time.Second {
		t.Errorf("expected 3s, got %v", backoff)
	}
}
