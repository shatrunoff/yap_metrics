package agent

import (
	"context"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/model"
	pb "github.com/shatrunoff/yap_metrics/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// GRPCSender отправляет метрики по gRPC.
type GRPCSender struct {
	conn   *grpc.ClientConn
	client pb.MetricsClient
}

// NewGRPCSender создаёт gRPC клиент для отправки метрик.
func NewGRPCSender(addr string) (*GRPCSender, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPCSender{conn: conn, client: pb.NewMetricsClient(conn)}, nil
}

// Close закрывает соединение.
func (s *GRPCSender) Close() error {
	return s.conn.Close()
}

// SendBatch отправляет батч метрик по gRPC.
func (s *GRPCSender) SendBatch(metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	pbMetrics := make([]*pb.Metric, 0, len(metrics))
	for _, m := range metrics {
		pm := &pb.Metric{Id: m.ID}
		switch m.MType {
		case model.Gauge:
			if m.Value == nil {
				continue
			}
			pm.Type = pb.Metric_GAUGE
			pm.Value = *m.Value
		case model.Counter:
			if m.Delta == nil {
				continue
			}
			pm.Type = pb.Metric_COUNTER
			pm.Delta = *m.Delta
		default:
			continue
		}
		pbMetrics = append(pbMetrics, pm)
	}

	if len(pbMetrics) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Добавляем IP в метаданные
	if ip := getLocalIP(); ip != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", ip)
	}

	_, err := s.client.UpdateMetrics(ctx, &pb.UpdateMetricsRequest{Metrics: pbMetrics})
	return err
}
