package db

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool aliases the pgxpool type so callers don't import it directly.
type Pool = pgxpool.Pool

// Tx is a transactional handle.
type Tx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgx.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type IndexerState struct {
	Contract      string
	LastBlock     uint64
	LastBlockHash common.Hash
}

func GetIndexerState(ctx context.Context, p *pgxpool.Pool, contract string) (*IndexerState, error) {
	row := p.QueryRow(ctx,
		"SELECT contract, last_block, last_block_hash FROM indexer_state WHERE contract=$1", contract)
	var s IndexerState
	var hash []byte
	if err := row.Scan(&s.Contract, &s.LastBlock, &hash); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	s.LastBlockHash = common.BytesToHash(hash)
	return &s, nil
}

func UpsertIndexerState(ctx context.Context, p *pgxpool.Pool, s IndexerState) error {
	_, err := p.Exec(ctx, `
        INSERT INTO indexer_state (contract, last_block, last_block_hash)
        VALUES ($1, $2, $3)
        ON CONFLICT (contract) DO UPDATE
          SET last_block = EXCLUDED.last_block,
              last_block_hash = EXCLUDED.last_block_hash,
              updated_at = now()`,
		s.Contract, s.LastBlock, s.LastBlockHash.Bytes())
	return err
}

func RecordBlock(ctx context.Context, p *pgxpool.Pool, num uint64, hash, parent common.Hash) error {
	_, err := p.Exec(ctx, `
        INSERT INTO block_trail (block_number, block_hash, parent_hash)
        VALUES ($1, $2, $3)
        ON CONFLICT (block_number) DO UPDATE
          SET block_hash = EXCLUDED.block_hash,
              parent_hash = EXCLUDED.parent_hash`,
		num, hash.Bytes(), parent.Bytes())
	return err
}

// PruneBlocksFrom rolls back state from `fromBlock` (inclusive) onwards.
// Used after reorg detection.
func PruneBlocksFrom(ctx context.Context, p *pgxpool.Pool, fromBlock uint64) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, table := range []string{"deposits", "withdrawals", "reward_distributions"} {
		if _, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE block_number >= $1", fromBlock); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, "DELETE FROM block_trail WHERE block_number >= $1", fromBlock); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type DepositEvent struct {
	TxHash      common.Hash
	LogIndex    uint
	BlockNumber uint64
	BlockHash   common.Hash
	Sender      common.Address
	Owner       common.Address
	Assets      *big.Int
	Shares      *big.Int
	Timestamp   time.Time
}

func InsertDeposit(ctx context.Context, p *pgxpool.Pool, e DepositEvent) error {
	_, err := p.Exec(ctx, `
        INSERT INTO deposits (tx_hash, log_index, block_number, block_hash,
                              sender, owner, assets, shares, timestamp)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        ON CONFLICT (tx_hash, log_index) DO NOTHING`,
		e.TxHash.Bytes(), e.LogIndex, e.BlockNumber, e.BlockHash.Bytes(),
		e.Sender.Bytes(), e.Owner.Bytes(), e.Assets.String(), e.Shares.String(), e.Timestamp)
	return err
}

type WithdrawEvent struct {
	TxHash      common.Hash
	LogIndex    uint
	BlockNumber uint64
	BlockHash   common.Hash
	Sender      common.Address
	Receiver    common.Address
	Owner       common.Address
	Assets      *big.Int
	Shares      *big.Int
	Timestamp   time.Time
}

func InsertWithdraw(ctx context.Context, p *pgxpool.Pool, e WithdrawEvent) error {
	_, err := p.Exec(ctx, `
        INSERT INTO withdrawals (tx_hash, log_index, block_number, block_hash,
                                 sender, receiver, owner, assets, shares, timestamp)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (tx_hash, log_index) DO NOTHING`,
		e.TxHash.Bytes(), e.LogIndex, e.BlockNumber, e.BlockHash.Bytes(),
		e.Sender.Bytes(), e.Receiver.Bytes(), e.Owner.Bytes(),
		e.Assets.String(), e.Shares.String(), e.Timestamp)
	return err
}

type RewardEvent struct {
	TxHash           common.Hash
	LogIndex         uint
	BlockNumber      uint64
	BlockHash        common.Hash
	Operator         common.Address
	Amount           *big.Int
	TotalAssetsAfter *big.Int
	Timestamp        time.Time
}

func InsertReward(ctx context.Context, p *pgxpool.Pool, e RewardEvent) error {
	_, err := p.Exec(ctx, `
        INSERT INTO reward_distributions (tx_hash, log_index, block_number, block_hash,
                                          operator, amount, total_assets_after, timestamp)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (tx_hash, log_index) DO NOTHING`,
		e.TxHash.Bytes(), e.LogIndex, e.BlockNumber, e.BlockHash.Bytes(),
		e.Operator.Bytes(), e.Amount.String(), e.TotalAssetsAfter.String(), e.Timestamp)
	return err
}
