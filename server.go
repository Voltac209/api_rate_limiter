package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"

	ratelimiterpb "github.com/Voltac209/api_rate_limiter/gen/ratelimiter/v1"
	ratelimiter "github.com/Voltac209/api_rate_limiter/src"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type RateLimiterServer struct {
	ratelimiterpb.UnimplementedRateLimiterServer
	limiter ratelimiter.Limiter
}

func (s *RateLimiterServer) Check(
	ctx context.Context,
	req *ratelimiterpb.RateLimitRequest,
) (*ratelimiterpb.RateLimitResponse, error) {
	if req.Key == "" {
		return nil, status.Error(codes.InvalidArgument, "key cannot be empty")
	}
	if req.Limit == 0 {
		return nil, status.Error(codes.InvalidArgument, "limit must be greater than 0")
	}
	if req.WindowSeconds == 0 {
		return nil, status.Error(codes.InvalidArgument, "windowSeconds must be greater than 0")
	}
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
	backend := strings.ToLower(os.Getenv("LIMITER_BACKEND"))

	if (backend=="") {
		backend="inmemory"
	}

	var limiter ratelimiter.Limiter
	switch backend {
	case "inmemory" :
		limiter=ratelimiter.NewTokenBucketLimiter()
		log.Println("Using In memory Token Bucket\n")

	case "redis":
		log.Println("Using Redis\n")
	
	default:
		log.Println("Invalid env use either inmemory or redis")
	}

	ratelimiterpb.RegisterRateLimiterServer(
		grpcServer,
		&RateLimiterServer{
			limiter: limiter,
		},
	)

	log.Println("RateLimiter gRPC server running on :50051")
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to server: %v", err)
	}

}
