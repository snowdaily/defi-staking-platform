package indexer

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"

	"github.com/garyfang/defi-staking-platform/backend/internal/chain"
	"github.com/garyfang/defi-staking-platform/backend/internal/db"
)

const contractKey = "staking_vault"

// Metrics exposed for Prometheus scrape.
var (
	blocksBehind = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "indexer_blocks_behind",
		Help: "How many blocks the indexer is behind the chain tip.",
	})
	eventsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "indexer_events_processed_total",
		Help: "Total events processed, by type.",
	}, []string{"event"})
	reorgs = promauto.NewCounter(prometheus.CounterOpts{
		Name: "indexer_reorgs_total",
		Help: "Total number of reorgs detected.",
	})
)

type Indexer struct {
	cfg   Config
	chain *chain.Client
	pool  *pgxpool.Pool
}

type Config struct {
	StartBlock   uint64
	BlockBatch   uint64
	ReorgDepth   uint64
	PollInterval time.Duration
}

func New(cfg Config, c *chain.Client, pool *pgxpool.Pool) *Indexer {
	return &Indexer{cfg: cfg, chain: c, pool: pool}
}

// Run is the indexer main loop.
//
// It catches up from the last persisted block to the current head in batches,
// then sleeps PollInterval between iterations. Reorg detection runs every
// iteration over the trailing ReorgDepth blocks.
func (i *Indexer) Run(ctx context.Context) error {
	state, err := db.GetIndexerState(ctx, i.pool, contractKey)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	cursor := i.cfg.StartBlock
	if state != nil {
		cursor = state.LastBlock + 1
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := i.detectReorg(ctx); err != nil {
			log.Error().Err(err).Msg("reorg check failed")
		}

		head, err := i.chain.Eth.BlockNumber(ctx)
		if err != nil {
			log.Error().Err(err).Msg("get head failed")
			i.sleep(ctx)
			continue
		}
		blocksBehind.Set(float64(head) - float64(cursor))

		if cursor > head {
			i.sleep(ctx)
			continue
		}

		end := cursor + i.cfg.BlockBatch - 1
		if end > head {
			end = head
		}

		if err := i.processRange(ctx, cursor, end); err != nil {
			log.Error().Err(err).Uint64("from", cursor).Uint64("to", end).Msg("process range failed")
			i.sleep(ctx)
			continue
		}
		cursor = end + 1
	}
}

func (i *Indexer) sleep(ctx context.Context) {
	t := time.NewTimer(i.cfg.PollInterval)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// processRange filters logs in [from, to] and ingests them.
func (i *Indexer) processRange(ctx context.Context, from, to uint64) error {
	q := i.chain.FilterQuery(new(big.Int).SetUint64(from), new(big.Int).SetUint64(to))
	logs, err := i.chain.Eth.FilterLogs(ctx, q)
	if err != nil {
		return err
	}

	for _, lg := range logs {
		if err := i.ingestLog(ctx, lg); err != nil {
			return err
		}
	}

	// Record block-trail entries so we can detect reorgs later.
	for n := from; n <= to; n++ {
		hdr, err := i.chain.HeaderAt(ctx, n)
		if err != nil {
			return err
		}
		if err := db.RecordBlock(ctx, i.pool, n, hdr.Hash(), hdr.ParentHash); err != nil {
			return err
		}
	}

	if to > 0 {
		hdr, err := i.chain.HeaderAt(ctx, to)
		if err != nil {
			return err
		}
		if err := db.UpsertIndexerState(ctx, i.pool, db.IndexerState{
			Contract:      contractKey,
			LastBlock:     to,
			LastBlockHash: hdr.Hash(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (i *Indexer) ingestLog(ctx context.Context, lg types.Log) error {
	hdr, err := i.chain.HeaderAt(ctx, lg.BlockNumber)
	if err != nil {
		return err
	}
	ts := time.Unix(int64(hdr.Time), 0).UTC()

	deposit := i.chain.ABI.Events["Deposit"].ID
	withdraw := i.chain.ABI.Events["Withdraw"].ID
	reward := i.chain.ABI.Events["RewardsDistributed"].ID

	switch lg.Topics[0] {
	case deposit:
		ev, err := decodeDeposit(i.chain, lg)
		if err != nil {
			return err
		}
		ev.Timestamp = ts
		eventsProcessed.WithLabelValues("deposit").Inc()
		return db.InsertDeposit(ctx, i.pool, ev)
	case withdraw:
		ev, err := decodeWithdraw(i.chain, lg)
		if err != nil {
			return err
		}
		ev.Timestamp = ts
		eventsProcessed.WithLabelValues("withdraw").Inc()
		return db.InsertWithdraw(ctx, i.pool, ev)
	case reward:
		ev, err := decodeReward(i.chain, lg)
		if err != nil {
			return err
		}
		ev.Timestamp = ts
		eventsProcessed.WithLabelValues("reward").Inc()
		return db.InsertReward(ctx, i.pool, ev)
	}
	return nil
}

// detectReorg walks backwards over the trail and rewinds state if a divergence
// is found.
func (i *Indexer) detectReorg(ctx context.Context) error {
	state, err := db.GetIndexerState(ctx, i.pool, contractKey)
	if err != nil || state == nil {
		return err
	}

	from := state.LastBlock
	depth := i.cfg.ReorgDepth
	if from < depth {
		return nil
	}
	to := from
	from = to - depth + 1

	for n := to; n >= from; n-- {
		hdr, err := i.chain.HeaderAt(ctx, n)
		if err != nil {
			return err
		}
		var stored []byte
		err = i.pool.QueryRow(ctx, "SELECT block_hash FROM block_trail WHERE block_number=$1", n).Scan(&stored)
		if err != nil {
			return nil // no trail at this depth — nothing to compare
		}
		if hdr.Hash() != common.BytesToHash(stored) {
			reorgs.Inc()
			log.Warn().Uint64("block", n).Msg("reorg detected — rewinding")
			if err := db.PruneBlocksFrom(ctx, i.pool, n); err != nil {
				return err
			}
			// Move cursor back so next loop re-ingests from `n`.
			return db.UpsertIndexerState(ctx, i.pool, db.IndexerState{
				Contract:      contractKey,
				LastBlock:     n - 1,
				LastBlockHash: common.Hash{},
			})
		}
	}
	return nil
}
