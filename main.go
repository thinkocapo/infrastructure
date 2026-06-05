package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	sentry "github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"

	"github.com/thinkocapo/infrastructure/collectors"
)

func main() {
	_ = godotenv.Load()

	// Which sources to run. -collectors flag wins, then COLLECTORS env, then all.
	//   go run . -collectors=host          → host only
	//   go run . -collectors=docker        → docker only
	//   go run . -collectors=host,docker   → both (same as default)
	//   go run .                           → all registered collectors
	collectorsFlag := flag.String("collectors", "",
		fmt.Sprintf("comma-separated sources to run (%s); empty = all", collectors.Names()))
	flag.Parse()

	selection := *collectorsFlag
	if selection == "" {
		selection = os.Getenv("COLLECTORS")
	}

	chosen, unknown := collectors.Select(collectors.ParseSelection(selection))
	if len(unknown) > 0 {
		log.Fatalf("unknown collector(s): %s — available: %s", strings.Join(unknown, ", "), collectors.Names())
	}
	if len(chosen) == 0 {
		log.Fatalf("no collectors selected — available: %s", collectors.Names())
	}

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

	active := make([]string, len(chosen))
	for i, c := range chosen {
		active[i] = c.Name
	}
	fmt.Printf("Starting infrastructure monitor — collectors: [%s] — emitting every %ds\n\n",
		strings.Join(active, ", "), interval)

	ctx := context.Background()

	for {
		fmt.Printf("[%s] collecting metrics...\n", time.Now().Format("15:04:05"))
		for _, c := range chosen {
			c.Collect(ctx)
		}
		fmt.Println()
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
