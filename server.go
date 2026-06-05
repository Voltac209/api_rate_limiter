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
	limiter     ratelimiter.Limiter
	configStore ratelimiter.ConfigStore
}

func (s *RateLimiterServer) Check(
	ctx context.Context,
	req *ratelimiterpb.RateLimitRequest,
) (*ratelimiterpb.RateLimitResponse, error) {
	if req.Key == "" {
		return nil, status.Error(codes.InvalidArgument, "key cannot be empty")
	}
	limit := req.Limit
	windowSeconds := req.WindowSeconds

	if s.configStore != nil {
		config, found, err := s.configStore.Get(ctx, req.Key)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to load rate limit configuration")
		}
		if found {
			limit = config.Limit
			windowSeconds = config.WindowSeconds
		}
	}
	if limit == 0 {
		return nil, status.Error(codes.InvalidArgument, "limit must be greater than 0")
	}
	if windowSeconds == 0 {
		return nil, status.Error(codes.InvalidArgument, "windowSeconds must be greater than 0")
	}
	result := s.limiter.Check(
		req.Key,
		limit,
		windowSeconds,
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
	algorithm := strings.ToLower(os.Getenv("LIMITER_ALGORITHM"))
	databaseURL := os.Getenv("DATABASE_URL")
	var configStore ratelimiter.ConfigStore
	if databaseURL != "" {
		store, err := ratelimiter.NewPostgresConfigStore(context.Background(), databaseURL)
		if err != nil {
			log.Fatalf("failed to connect to PostgreSQL: %v", err)
		}
		defer store.Close()

		configStore = store
		log.Println("Using stored configuration")
	}

	if backend == "" {
		backend = "inmemory"
	}
	if algorithm == "" {
		algorithm = "token_bucket"
	}

	var limiter ratelimiter.Limiter
	switch backend {
	case "inmemory":
		switch algorithm {
		case "token_bucket":
			limiter = ratelimiter.NewTokenBucketLimiter()
			log.Println("Using in-memory token bucket limiter")
		case "rolling_window":
			limiter = ratelimiter.NewRollingWindowLimiter()
			log.Println("Using in-memory rolling window limiter")
		default:
			log.Fatalf("invalid LIMITER_ALGORITHM=%q; use token_bucket or rolling_window", algorithm)
		}

	case "redis":
		redisAddr := os.Getenv("REDIS_ADDR")
		if redisAddr == "" {
			log.Fatal("REDIS_ADDR required when LIMITER_BACKEND=redis")
		}
		if algorithm != "token_bucket" && algorithm != "rolling_window" {
			log.Fatalf("invalid LIMITER_ALGORITHM=%q; use token_bucket or rolling_window", algorithm)
		}
		limiter = ratelimiter.NewRedisLimiter(redisAddr, algorithm)
		log.Printf("Using Redis %s limiter backend (%s)", algorithm, redisAddr)
	default:
		log.Fatalf("invalid LIMITER_BACKEND=%q; use inmemory or redis", backend)
	}

	ratelimiterpb.RegisterRateLimiterServer(
		grpcServer,
		&RateLimiterServer{
			limiter:     limiter,
			configStore: configStore,
		},
	)

	log.Println("RateLimiter gRPC server running on :50051")
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to server: %v", err)
	}

}
