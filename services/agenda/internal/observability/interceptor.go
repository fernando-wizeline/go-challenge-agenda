package observability

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryLoggingInterceptor logs every unary RPC with its method, status code and
// duration. Server-side failures are logged as errors, everything else as info.
func UnaryLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		code := status.Code(err)
		attrs := []any{"method", info.FullMethod, "code", code.String(), "duration", time.Since(start)}
		switch code {
		case codes.Internal, codes.Unknown, codes.DataLoss, codes.Unavailable:
			logger.ErrorContext(ctx, "rpc failed", append(attrs, "error", err)...)
		case codes.OK:
			logger.InfoContext(ctx, "rpc completed", attrs...)
		default:
			logger.InfoContext(ctx, "rpc completed", append(attrs, "error", err)...)
		}
		return resp, err
	}
}
