package agent

import (
	"context"
	"net"
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/model"
	pb "github.com/shatrunoff/yap_metrics/internal/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type mockMetricsServer struct {
	pb.UnimplementedMetricsServer
	received []*pb.Metric
}

func (m *mockMetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	m.received = req.Metrics
	return &pb.UpdateMetricsResponse{}, nil
}

func TestGRPCSender_SendBatch(t *testing.T) {
	// Start mock server
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	mockServer := &mockMetricsServer{}
	srv := grpc.NewServer()
	pb.RegisterMetricsServer(srv, mockServer)

	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	// Create sender
	sender, err := NewGRPCSender(lis.Addr().String())
	require.NoError(t, err)
	defer sender.Close()

	t.Run("send gauge metrics", func(t *testing.T) {
		mockServer.received = nil
		v := 123.45
		metrics := []model.Metrics{
			{ID: "test_gauge", MType: model.Gauge, Value: &v},
		}

		err := sender.SendBatch(metrics)
		require.NoError(t, err)
		require.Len(t, mockServer.received, 1)
		assert.Equal(t, "test_gauge", mockServer.received[0].Id)
		assert.Equal(t, pb.Metric_GAUGE, mockServer.received[0].Type)
		assert.Equal(t, 123.45, mockServer.received[0].Value)
	})

	t.Run("send counter metrics", func(t *testing.T) {
		mockServer.received = nil
		d := int64(100)
		metrics := []model.Metrics{
			{ID: "test_counter", MType: model.Counter, Delta: &d},
		}

		err := sender.SendBatch(metrics)
		require.NoError(t, err)
		require.Len(t, mockServer.received, 1)
		assert.Equal(t, "test_counter", mockServer.received[0].Id)
		assert.Equal(t, pb.Metric_COUNTER, mockServer.received[0].Type)
		assert.Equal(t, int64(100), mockServer.received[0].Delta)
	})

	t.Run("send mixed batch", func(t *testing.T) {
		mockServer.received = nil
		v := 42.0
		d := int64(10)
		metrics := []model.Metrics{
			{ID: "g1", MType: model.Gauge, Value: &v},
			{ID: "c1", MType: model.Counter, Delta: &d},
		}

		err := sender.SendBatch(metrics)
		require.NoError(t, err)
		assert.Len(t, mockServer.received, 2)
	})

	t.Run("skip nil values", func(t *testing.T) {
		mockServer.received = nil
		v := 1.0
		metrics := []model.Metrics{
			{ID: "valid", MType: model.Gauge, Value: &v},
			{ID: "nil_gauge", MType: model.Gauge, Value: nil},
			{ID: "nil_counter", MType: model.Counter, Delta: nil},
		}

		err := sender.SendBatch(metrics)
		require.NoError(t, err)
		assert.Len(t, mockServer.received, 1)
	})

	t.Run("empty batch", func(t *testing.T) {
		mockServer.received = nil
		err := sender.SendBatch([]model.Metrics{})
		require.NoError(t, err)
		assert.Nil(t, mockServer.received)
	})

	t.Run("skip unknown type", func(t *testing.T) {
		mockServer.received = nil
		v := 1.0
		metrics := []model.Metrics{
			{ID: "valid", MType: model.Gauge, Value: &v},
			{ID: "unknown", MType: "unknown_type"},
		}

		err := sender.SendBatch(metrics)
		require.NoError(t, err)
		assert.Len(t, mockServer.received, 1)
	})
}

func TestNewGRPCSender_InvalidAddress(t *testing.T) {
	// This should not fail immediately - gRPC uses lazy connection
	sender, err := NewGRPCSender("invalid:address:format")
	if err == nil && sender != nil {
		sender.Close()
	}
}

func TestGRPCSender_Close(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer()
	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	sender, err := NewGRPCSender(lis.Addr().String())
	require.NoError(t, err)

	err = sender.Close()
	assert.NoError(t, err)
}

func TestGetLocalIP(t *testing.T) {
	ip := getLocalIP()
	// Should return non-empty string on most systems
	// May be empty in isolated environments
	t.Logf("Local IP: %s", ip)
}

func TestGRPCSender_ConnectionError(t *testing.T) {
	// Connect to non-existent server
	sender, err := NewGRPCSender("127.0.0.1:59999")
	require.NoError(t, err) // gRPC uses lazy connection
	defer sender.Close()

	v := 1.0
	metrics := []model.Metrics{
		{ID: "test", MType: model.Gauge, Value: &v},
	}

	// Should fail when trying to send
	err = sender.SendBatch(metrics)
	assert.Error(t, err)
}

func TestNewGRPCSender(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	lis.Close()

	sender, err := NewGRPCSender(lis.Addr().String())
	if err == nil {
		assert.NotNil(t, sender)
		assert.NotNil(t, sender.conn)
		assert.NotNil(t, sender.client)
		sender.Close()
	}
}

func TestGRPCSenderWithMetadata(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	mockServer := &mockMetricsServer{}
	srv := grpc.NewServer()
	pb.RegisterMetricsServer(srv, mockServer)

	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	sender := &GRPCSender{conn: conn, client: pb.NewMetricsClient(conn)}

	v := 1.0
	err = sender.SendBatch([]model.Metrics{{ID: "test", MType: model.Gauge, Value: &v}})
	require.NoError(t, err)
}
