package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	sentry "github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"

	"github.com/thinkocapo/infrastructure/collectors"
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Fatal("SENTRY_DSN is required")
	}

	interval := 60
	if v := os.Getenv("INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			interval = n
		}
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		TracesSampleRate: 0.0,
	}); err != nil {
		log.Fatalf("sentry.Init: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	fmt.Printf("Starting infrastructure monitor — emitting every %ds\n\n", interval)

	ctx := context.Background()

	for {
		fmt.Printf("[%s] collecting metrics...\n", time.Now().Format("15:04:05"))
		collectors.CollectHost(ctx)
		collectors.CollectDocker(ctx)
		fmt.Println()
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
