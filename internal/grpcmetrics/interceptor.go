package grpcmetrics

import (
	"context"
	"net"
	"strings"

	pb "github.com/Radiushina/metrics-receiver.git/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TrustedSubnetInterceptor проверяет IP агента из метаданных x-real-ip.
// Если network == nil (пустой trusted_subnet), проверка не выполняется.
func TrustedSubnetInterceptor(network *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if network == nil {
			return handler(ctx, req)
		}
		ip := net.ParseIP(strings.TrimSpace(realIPFromMD(ctx)))
		if ip == nil || !network.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}
		return handler(ctx, req)
	}
}

func realIPFromMD(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get(pb.RealIPMetadataKey)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
