package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/asmisnik/users-notifier/internal/config"
	"github.com/asmisnik/users-notifier/internal/handler"
	_ "github.com/asmisnik/users-notifier/internal/metrics"
	"github.com/asmisnik/users-notifier/internal/telegram"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	logger, loggerCleanup, err := buildLogger(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "building logger: %v\n", err)
		os.Exit(1)
	}
	defer loggerCleanup()
	defer logger.Sync() //nolint:errcheck

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	tg := telegram.NewClient(cfg.Telegram.BotToken)
	h := handler.New(tg, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/send", h.Send)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	apiServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: mux,
	}

	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Metrics.Port),
		Handler: promhttp.Handler(),
	}

	go func() {
		logger.Info("api server listening", zap.Int("port", cfg.Server.Port))
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("api server error", zap.Error(err))
		}
	}()
	go func() {
		logger.Info("metrics server listening", zap.Int("port", cfg.Metrics.Port))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server error", zap.Error(err))
		}
	}()

	<-ctx.Done()

	bg := context.Background()
	_ = apiServer.Shutdown(bg)
	_ = metricsServer.Shutdown(bg)
	logger.Info("shutdown complete")
}

func buildLogger(cfg config.LoggingConfig) (*zap.Logger, func(), error) {
	level, err := zap.ParseAtomicLevel(cfg.Level)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(os.Stdout),
		level,
	)

	cores := []zapcore.Core{stdoutCore}
	cleanup := func() {}

	if cfg.FilePath != "" {
		if err := os.MkdirAll(dirOf(cfg.FilePath), 0755); err != nil {
			return nil, nil, fmt.Errorf("creating log dir: %w", err)
		}
		f, err := os.OpenFile(cfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("opening log file %s: %w", cfg.FilePath, err)
		}
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(f),
			level,
		)
		cores = append(cores, fileCore)
		cleanup = func() { f.Close() }
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller()), cleanup, nil
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
