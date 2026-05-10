package main

import (
	"context"
	"fmt"
	"log"
	"time"

	ratelimiterpb "github.com/Voltac209/api_rate_limiter/gen/ratelimiter/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to dial server: %v", err)
	}
	defer conn.Close()

	client := ratelimiterpb.NewRateLimiterClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.Check(ctx, &ratelimiterpb.RateLimitRequest{
		Key:           "demo-client",
		Limit:         3,
		WindowSeconds: 10,
	})
	if err != nil {
		log.Fatalf("check call failed: %v", err)
	}

	fmt.Printf("decision=%s remaining=%d retry_after_ms=%d\n", resp.Decision.String(), resp.Remaining, resp.RetryAfterMs)
}
