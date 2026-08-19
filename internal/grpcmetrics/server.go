// Package grpcmetrics реализует gRPC-сервис Metrics и interceptor доверенной подсети.
package grpcmetrics

import (
	"context"
	"net"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/audit"
	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	pb "github.com/Radiushina/metrics-receiver.git/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	// MetricsUpdater сохраняет пакет метрик.
	MetricsUpdater interface {
		UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error
	}

	// Saver сохраняет метрики на диск после обновления, если включено синхронное сохранение.
	Saver interface {
		Save(ctx context.Context) error
	}

	// AuditPublisher рассылает события аудита после успешного приёма метрик.
	AuditPublisher interface {
		Enabled() bool
		Notify(ctx context.Context, e audit.Event)
	}

	// Server реализует pb.MetricsServer.
	Server struct {
		pb.UnimplementedMetricsServer
		svc   MetricsUpdater
		saver Saver
		log   *zap.Logger
		audit AuditPublisher
	}
)

// NewServer создаёт gRPC-обработчик метрик.
func NewServer(svc MetricsUpdater, saver Saver, log *zap.Logger, auditPub AuditPublisher) *Server {
	return &Server{
		svc:   svc,
		saver: saver,
		log:   logger.OrNop(log),
		audit: auditPub,
	}
}

// NewGRPCServer собирает *grpc.Server с UnaryInterceptor проверки trusted_subnet.
func NewGRPCServer(impl pb.MetricsServer, trustedNet *net.IPNet) *grpc.Server {
	s := grpc.NewServer(grpc.UnaryInterceptor(TrustedSubnetInterceptor(trustedNet)))
	pb.RegisterMetricsServer(s, impl)
	return s
}

// UpdateMetrics принимает батч метрик и сохраняет их.
func (s *Server) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	if req == nil {
		return &pb.UpdateMetricsResponse{}, nil
	}

	in, err := pb.FromProto(req.GetMetrics())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(in) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	if err := s.svc.UpdateMetricsBatch(ctx, in); err != nil {
		s.log.Error("grpc update metrics batch", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal server error")
	}

	if s.saver != nil {
		if err := s.saver.Save(ctx); err != nil {
			s.log.Error("grpc persist metrics", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to persist metrics")
		}
	}

	s.notifyAudit(ctx, in)
	return &pb.UpdateMetricsResponse{}, nil
}

func (s *Server) notifyAudit(ctx context.Context, in []models.Metrics) {
	if s.audit == nil || !s.audit.Enabled() {
		return
	}
	names := make([]string, len(in))
	for i, m := range in {
		names[i] = m.ID
	}
	s.audit.Notify(ctx, audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   names,
		IPAddress: realIPFromMD(ctx),
	})
}
