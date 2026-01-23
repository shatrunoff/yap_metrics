package grpcserver

import (
	"context"
	"net"
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/model"
	pb "github.com/shatrunoff/yap_metrics/internal/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type mockStorage struct {
	metrics []model.Metrics
}

func (m *mockStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	return nil
}
func (m *mockStorage) UpdateCounter(ctx context.Context, name string, delta int64) error {
	return nil
}
func (m *mockStorage) GetMetric(ctx context.Context, mtype, name string) (model.Metrics, error) {
	return model.Metrics{}, nil
}
func (m *mockStorage) GetAll(ctx context.Context) (map[string]model.Metrics, error) {
	return nil, nil
}
func (m *mockStorage) UpdateMetricsBatch(ctx context.Context, metrics []model.Metrics) error {
	m.metrics = metrics
	return nil
}
func (m *mockStorage) Ping(ctx context.Context) error { return nil }
func (m *mockStorage) Close() error                   { return nil }

func TestMetricsServer_UpdateMetrics(t *testing.T) {
	storage := &mockStorage{}
	srv := NewMetricsServer(storage)

	tests := []struct {
		name    string
		req     *pb.UpdateMetricsRequest
		wantLen int
	}{
		{
			name: "gauge metric",
			req: &pb.UpdateMetricsRequest{
				Metrics: []*pb.Metric{
					{Id: "test_gauge", Type: pb.Metric_GAUGE, Value: 123.45},
				},
			},
			wantLen: 1,
		},
		{
			name: "counter metric",
			req: &pb.UpdateMetricsRequest{
				Metrics: []*pb.Metric{
					{Id: "test_counter", Type: pb.Metric_COUNTER, Delta: 100},
				},
			},
			wantLen: 1,
		},
		{
			name: "batch metrics",
			req: &pb.UpdateMetricsRequest{
				Metrics: []*pb.Metric{
					{Id: "g1", Type: pb.Metric_GAUGE, Value: 1.1},
					{Id: "c1", Type: pb.Metric_COUNTER, Delta: 10},
				},
			},
			wantLen: 2,
		},
		{
			name:    "empty request",
			req:     &pb.UpdateMetricsRequest{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage.metrics = nil
			resp, err := srv.UpdateMetrics(context.Background(), tt.req)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Len(t, storage.metrics, tt.wantLen)
		})
	}
}

func TestTrustedSubnetInterceptor(t *testing.T) {
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	tests := []struct {
		name          string
		trustedSubnet string
		ip            string
		wantErr       bool
		errCode       codes.Code
	}{
		{
			name:          "empty subnet allows all",
			trustedSubnet: "",
			ip:            "",
			wantErr:       false,
		},
		{
			name:          "invalid CIDR allows all",
			trustedSubnet: "invalid",
			ip:            "",
			wantErr:       false,
		},
		{
			name:          "missing x-real-ip",
			trustedSubnet: "192.168.0.0/16",
			ip:            "",
			wantErr:       true,
			errCode:       codes.PermissionDenied,
		},
		{
			name:          "IP in subnet",
			trustedSubnet: "192.168.0.0/16",
			ip:            "192.168.1.100",
			wantErr:       false,
		},
		{
			name:          "IP not in subnet",
			trustedSubnet: "192.168.0.0/16",
			ip:            "10.0.0.1",
			wantErr:       true,
			errCode:       codes.PermissionDenied,
		},
		{
			name:          "invalid IP",
			trustedSubnet: "192.168.0.0/16",
			ip:            "invalid-ip",
			wantErr:       true,
			errCode:       codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := TrustedSubnetInterceptor(tt.trustedSubnet)
			ctx := context.Background()
			if tt.ip != "" {
				md := metadata.Pairs("x-real-ip", tt.ip)
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.errCode, st.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, "ok", resp)
			}
		})
	}
}

func TestNewGRPCServer(t *testing.T) {
	storage := &mockStorage{}

	t.Run("without trusted subnet", func(t *testing.T) {
		srv := NewGRPCServer(storage, "")
		assert.NotNil(t, srv)
	})

	t.Run("with trusted subnet", func(t *testing.T) {
		srv := NewGRPCServer(storage, "192.168.0.0/16")
		assert.NotNil(t, srv)
	})
}

func TestGRPCServerIntegration(t *testing.T) {
	storage := &mockStorage{}
	srv := NewGRPCServer(storage, "")

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewMetricsClient(conn)

	resp, err := client.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{
			{Id: "test", Type: pb.Metric_GAUGE, Value: 42.0},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, storage.metrics, 1)
}
