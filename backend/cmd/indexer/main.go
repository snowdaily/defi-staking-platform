package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/garyfang/defi-staking-platform/backend/internal/chain"
	"github.com/garyfang/defi-staking-platform/backend/internal/config"
	"github.com/garyfang/defi-staking-platform/backend/internal/db"
	"github.com/garyfang/defi-staking-platform/backend/internal/indexer"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}
	if err := cfg.RequireForIndexer(); err != nil {
		log.Fatal().Err(err).Msg("config")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db")
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatal().Err(err).Msg("migrate")
	}

	c, err := chain.NewClient(ctx, cfg.RPCHTTPURL, common.HexToAddress(cfg.VaultAddress))
	if err != nil {
		log.Fatal().Err(err).Msg("chain client")
	}

	go startMetricsServer(cfg.MetricsAddr)

	idx := indexer.New(indexer.Config{
		StartBlock:   cfg.StartBlock,
		BlockBatch:   cfg.BlockBatch,
		ReorgDepth:   cfg.ReorgDepth,
		PollInterval: cfg.PollInterval,
	}, c, pool)

	log.Info().Str("vault", cfg.VaultAddress).Msg("indexer starting")
	if err := idx.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal().Err(err).Msg("indexer terminated")
	}
}

func startMetricsServer(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("metrics server")
	}
}
