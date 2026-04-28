package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the unified runtime configuration for all backend services.
// Loaded from environment variables; missing required fields produce errors.
type Config struct {
	// Database
	DatabaseURL string

	// Chain
	RPCHTTPURL    string
	RPCWSURL      string
	ChainID       int64
	StartBlock    uint64
	BlockBatch    uint64
	ReorgDepth    uint64
	PollInterval  time.Duration
	VaultAddress  string

	// Reward bot
	OperatorPrivateKey string
	RewardSchedule     string // cron expression
	MaxGasGwei         int64
	DryRun             bool

	// API
	ListenAddr string

	// Telemetry
	MetricsAddr string
	LogLevel    string
}

func Load() (*Config, error) {
	c := &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://staking:staking@localhost:5432/staking?sslmode=disable"),
		RPCHTTPURL:     getEnv("RPC_HTTP_URL", "http://localhost:8545"),
		RPCWSURL:       getEnv("RPC_WS_URL", "ws://localhost:8545"),
		ChainID:        getEnvInt64("CHAIN_ID", 31337),
		StartBlock:     uint64(getEnvInt64("START_BLOCK", 0)),
		BlockBatch:     uint64(getEnvInt64("BLOCK_BATCH", 1000)),
		ReorgDepth:     uint64(getEnvInt64("REORG_DEPTH", 64)),
		PollInterval:   getEnvDuration("POLL_INTERVAL", 4*time.Second),
		VaultAddress:   getEnv("VAULT_ADDRESS", ""),
		RewardSchedule: getEnv("REWARD_SCHEDULE", "0 */6 * * *"),
		MaxGasGwei:     getEnvInt64("MAX_GAS_GWEI", 100),
		DryRun:         getEnv("DRY_RUN", "false") == "true",
		ListenAddr:     getEnv("LISTEN_ADDR", ":8080"),
		MetricsAddr:    getEnv("METRICS_ADDR", ":9100"),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		// Operator key intentionally has no default — must be supplied for prod use.
		OperatorPrivateKey: os.Getenv("OPERATOR_PRIVATE_KEY"),
	}
	return c, nil
}

// RequireForIndexer validates the fields the indexer cares about.
func (c *Config) RequireForIndexer() error {
	if c.VaultAddress == "" {
		return errors.New("VAULT_ADDRESS is required")
	}
	return nil
}

// RequireForRewardBot validates fields needed for signing transactions.
func (c *Config) RequireForRewardBot() error {
	if c.VaultAddress == "" {
		return errors.New("VAULT_ADDRESS is required")
	}
	if !c.DryRun && c.OperatorPrivateKey == "" {
		return errors.New("OPERATOR_PRIVATE_KEY is required (or set DRY_RUN=true)")
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt64(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("invalid int64 for %s: %v", key, err))
		}
		return n
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			panic(fmt.Sprintf("invalid duration for %s: %v", key, err))
		}
		return d
	}
	return def
}
