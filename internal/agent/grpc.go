package agent

import (
	"context"
	"strings"
	"time"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	pb "github.com/Radiushina/metrics-receiver.git/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const grpcRequestTimeout = 5 * time.Second

// DialMetrics устанавливает gRPC-соединение с сервером метрик.
func DialMetrics(addr string) (*grpc.ClientConn, pb.MetricsClient, error) {
	addr = strings.TrimSpace(addr)
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "http://")
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return conn, pb.NewMetricsClient(conn), nil
}

// PostMetricsBatchGRPC отправляет пакет метрик через gRPC UpdateMetrics.
// IP агента передаётся в метаданных x-real-ip.
func PostMetricsBatchGRPC(ctx context.Context, client pb.MetricsClient, localIP string, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	req := &pb.UpdateMetricsRequest{Metrics: pb.ToProto(metrics)}
	return retryAgent(func() (bool, bool, error) {
		callCtx, cancel := context.WithTimeout(ctx, grpcRequestTimeout)
		if ip := strings.TrimSpace(localIP); ip != "" {
			callCtx = metadata.NewOutgoingContext(callCtx, metadata.Pairs(pb.RealIPMetadataKey, ip))
		}
		_, err := client.UpdateMetrics(callCtx, req)
		cancel()
		if err == nil {
			return true, false, nil
		}
		return false, isRetriableGRPCErr(err), err
	})
}

func isRetriableGRPCErr(err error) bool {
	if err == nil {
		return false
	}
	if isRetriableTransportErr(err) {
		return true
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
		return true
	default:
		return false
	}
}
