package grpcserver

import (
	"context"
	"net"

	"github.com/shatrunoff/yap_metrics/internal/model"
	pb "github.com/shatrunoff/yap_metrics/internal/proto"
	"github.com/shatrunoff/yap_metrics/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// MetricsServer реализует gRPC сервис метрик.
type MetricsServer struct {
	pb.UnimplementedMetricsServer
	storage storage.Storage
}

// NewMetricsServer создаёт новый gRPC сервер метрик.
func NewMetricsServer(st storage.Storage) *MetricsServer {
	return &MetricsServer{storage: st}
}

// UpdateMetrics обновляет метрики на сервере.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	metrics := make([]model.Metrics, 0, len(req.Metrics))
	for _, m := range req.Metrics {
		metric := model.Metrics{ID: m.Id}
		switch m.Type {
		case pb.Metric_GAUGE:
			metric.MType = model.Gauge
			v := m.Value
			metric.Value = &v
		case pb.Metric_COUNTER:
			metric.MType = model.Counter
			d := m.Delta
			metric.Delta = &d
		}
		metrics = append(metrics, metric)
	}

	if err := s.storage.UpdateMetricsBatch(ctx, metrics); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update metrics: %v", err)
	}

	return &pb.UpdateMetricsResponse{}, nil
}

// TrustedSubnetInterceptor проверяет IP-адрес клиента из метаданных x-real-ip.
func TrustedSubnetInterceptor(trustedSubnet string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if trustedSubnet == "" {
			return handler(ctx, req)
		}

		_, subnet, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing metadata")
		}

		ips := md.Get("x-real-ip")
		if len(ips) == 0 {
			return nil, status.Error(codes.PermissionDenied, "missing x-real-ip")
		}

		ip := net.ParseIP(ips[0])
		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "IP not in trusted subnet")
		}

		return handler(ctx, req)
	}
}

// NewGRPCServer создаёт и настраивает gRPC сервер.
func NewGRPCServer(st storage.Storage, trustedSubnet string) *grpc.Server {
	opts := []grpc.ServerOption{}
	if trustedSubnet != "" {
		opts = append(opts, grpc.UnaryInterceptor(TrustedSubnetInterceptor(trustedSubnet)))
	}

	srv := grpc.NewServer(opts...)
	pb.RegisterMetricsServer(srv, NewMetricsServer(st))
	return srv
}
