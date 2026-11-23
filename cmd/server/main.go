package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"unwind/internal/httpapi"
	"unwind/internal/plan"
	"unwind/internal/reply"
	"unwind/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseURL := env("DATABASE_URL", "postgresql:///unwind?host=/tmp")
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}
	database := store.New(pool)
	if err := database.Migrate(ctx); err != nil {
		return err
	}
	for _, code := range strings.Split(env("ACTIVATION_CODES", "UNWIND-DEMO"), ",") {
		if code = strings.TrimSpace(code); code != "" {
			if err := database.RegisterActivation(ctx, code); err != nil {
				return err
			}
		}
	}

	publicBeta := env("PUBLIC_BETA", "true") == "true"
	dailyReplyLimit, err := envInt("PUBLIC_DAILY_REPLY_LIMIT", 3)
	if err != nil {
		return err
	}
	aiDailyLimit, err := envInt("AI_DAILY_LIMIT", 100)
	if err != nil {
		return err
	}
	var generator plan.Generator
	if os.Getenv("AI_API_KEY") != "" {
		generator = reply.NewOpenAI(
			env("AI_BASE_URL", "https://api.openai.com/v1"),
			os.Getenv("AI_API_KEY"),
			env("AI_MODEL", "gpt-4o-mini"),
			nil,
		)
		generator = reply.NewDailyBudget(generator, aiDailyLimit)
	}
	generator = reply.NewSafe(generator)

	address := ":" + env("PORT", "8080")
	server := &http.Server{
		Addr: address,
		Handler: httpapi.NewRouterWithConfig(database, generator, os.DirFS(env("WEB_DIR", "web")), os.Getenv("COOKIE_SECURE") == "true", httpapi.Config{
			PublicBeta:      publicBeta,
			DailyReplyLimit: dailyReplyLimit,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownContext)
	}()
	log.Printf("unwind listening on %s", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}
