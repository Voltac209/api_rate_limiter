package main

import (
	"context"
	"log"
	"net"

	ratelimiterpb "github.com/Voltac209/api_rate_limiter/gen/ratelimiter/v1"
	ratelimiter "github.com/Voltac209/api_rate_limiter/src"
	"google.golang.org/grpc"
)

type RateLimiterServer struct {
	ratelimiterpb.UnimplementedRateLimiterServer
	limiter ratelimiter.Limiter
}

func (s *RateLimiterServer) Check(
	ctx context.Context,
	req *ratelimiterpb.RateLimitRequest,
) (*ratelimiterpb.RateLimitResponse, error) {
	result := s.limiter.Check(
		req.Key,
		req.Limit,
		req.WindowSeconds,
	)
	decision := ratelimiterpb.RateLimitResponse_DENY
	if result.Allowed {
		decision = *ratelimiterpb.RateLimitResponse_ALLOW.Enum()
	}
	return &ratelimiterpb.RateLimitResponse{
		Decision:     decision,
		Remaining:    result.Remaining,
		RetryAfterMs: result.RetryAfterMs,
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen : %v", err)
	}
	grpcServer := grpc.NewServer()
	limiter := ratelimiter.NewTokenBucketLimiter()

	ratelimiterpb.RegisterRateLimiterServer(
		grpcServer,
		&RateLimiterServer{
			limiter: limiter,
		},
	)

	log.Println("RateLimiter gRPC server running on :50051")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to server: %v", err)
	}
}
