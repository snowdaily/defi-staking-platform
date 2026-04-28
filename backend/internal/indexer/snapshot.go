package indexer

import (
	"context"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/garyfang/defi-staking-platform/backend/internal/chain"
)

// snapshotExchangeRate reads the vault's totalAssets / totalSupply at the
// current chain head and persists a snapshot row used by the /tvl and /apr
// endpoints. Idempotent: callers should rate-limit (e.g., once per processed
// range or once per N blocks) — this function does not deduplicate.
func snapshotExchangeRate(ctx context.Context, c *chain.Client, pool *pgxpool.Pool, blockNumber uint64, ts time.Time) error {
	totalAssets, err := c.TotalAssets(ctx)
	if err != nil {
		return err
	}
	totalSupply, err := c.TotalSupply(ctx)
	if err != nil {
		return err
	}

	// rate_e27 = totalAssets * 1e27 / totalSupply (when supply > 0).
	// Stored as a fixed-point share/asset rate so APR computation stays in integers.
	var rateE27 *big.Int
	if totalSupply.Sign() == 0 {
		rateE27 = big.NewInt(0)
	} else {
		rateE27 = new(big.Int).Mul(totalAssets, big.NewInt(1))
		// 1e27
		exp := new(big.Int).Exp(big.NewInt(10), big.NewInt(27), nil)
		rateE27.Mul(totalAssets, exp)
		rateE27.Quo(rateE27, totalSupply)
	}

	_, err = pool.Exec(ctx, `
        INSERT INTO exchange_rate_snapshots (block_number, timestamp, total_assets, total_supply, rate_e27)
        VALUES ($1, $2, $3, $4, $5)`,
		blockNumber, ts, totalAssets.String(), totalSupply.String(), rateE27.String())
	return err
}
