package main

import (
	"context"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/garyfang/defi-staking-platform/backend/internal/chain"
	"github.com/garyfang/defi-staking-platform/backend/internal/config"
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}
	if err := cfg.RequireForRewardBot(); err != nil {
		log.Fatal().Err(err).Msg("config")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	c, err := chain.NewClient(ctx, cfg.RPCHTTPURL, common.HexToAddress(cfg.VaultAddress))
	if err != nil {
		log.Fatal().Err(err).Msg("chain client")
	}

	var signer chain.Signer
	if !cfg.DryRun {
		signer, err = chain.NewEnvSigner(cfg.OperatorPrivateKey)
		if err != nil {
			log.Fatal().Err(err).Msg("signer")
		}
	}

	sender := &chain.TxSender{
		Client:     c,
		Signer:     signer,
		ChainID:    big.NewInt(cfg.ChainID),
		MaxGasGwei: cfg.MaxGasGwei,
	}

	// Reward bot v1: distribute a fixed amount each tick.
	// Production version would compute the right amount from the yield source.
	rewardAmount := new(big.Int).Mul(big.NewInt(10), big.NewInt(1_000_000_000_000_000_000)) // 10 tokens

	job := func() {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		log.Info().Msg("reward tick: building tx")

		data, err := c.ABI.Pack("distributeRewards", rewardAmount)
		if err != nil {
			log.Error().Err(err).Msg("pack")
			return
		}
		h, err := sender.SendCall(ctx, c.Vault, data, cfg.DryRun)
		if err != nil {
			log.Error().Err(err).Msg("send failed")
			return
		}
		if cfg.DryRun {
			return
		}
		r, err := sender.WaitForReceipt(ctx, h)
		if err != nil {
			log.Error().Err(err).Str("tx", h.Hex()).Msg("receipt failed")
			return
		}
		log.Info().Str("tx", h.Hex()).Uint64("block", r.BlockNumber.Uint64()).Msg("reward distributed")
	}

	cr := cron.New(cron.WithSeconds())
	if _, err := cr.AddFunc(cfg.RewardSchedule, job); err != nil {
		log.Fatal().Err(err).Msg("cron")
	}
	cr.Start()

	log.Info().
		Str("schedule", cfg.RewardSchedule).
		Bool("dryRun", cfg.DryRun).
		Msg("reward bot running")

	<-ctx.Done()
	stopCtx := cr.Stop()
	<-stopCtx.Done()
}
