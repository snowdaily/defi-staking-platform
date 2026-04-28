package indexer

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/garyfang/defi-staking-platform/backend/internal/chain"
	"github.com/garyfang/defi-staking-platform/backend/internal/db"
)

func decodeDeposit(c *chain.Client, lg types.Log) (db.DepositEvent, error) {
	if len(lg.Topics) < 3 {
		return db.DepositEvent{}, fmt.Errorf("deposit: too few topics")
	}
	values, err := c.ABI.Events["Deposit"].Inputs.NonIndexed().Unpack(lg.Data)
	if err != nil {
		return db.DepositEvent{}, err
	}
	return db.DepositEvent{
		TxHash:      lg.TxHash,
		LogIndex:    lg.Index,
		BlockNumber: lg.BlockNumber,
		BlockHash:   lg.BlockHash,
		Sender:      common.BytesToAddress(lg.Topics[1].Bytes()),
		Owner:       common.BytesToAddress(lg.Topics[2].Bytes()),
		Assets:      values[0].(*big.Int),
		Shares:      values[1].(*big.Int),
	}, nil
}

func decodeWithdraw(c *chain.Client, lg types.Log) (db.WithdrawEvent, error) {
	if len(lg.Topics) < 4 {
		return db.WithdrawEvent{}, fmt.Errorf("withdraw: too few topics")
	}
	values, err := c.ABI.Events["Withdraw"].Inputs.NonIndexed().Unpack(lg.Data)
	if err != nil {
		return db.WithdrawEvent{}, err
	}
	return db.WithdrawEvent{
		TxHash:      lg.TxHash,
		LogIndex:    lg.Index,
		BlockNumber: lg.BlockNumber,
		BlockHash:   lg.BlockHash,
		Sender:      common.BytesToAddress(lg.Topics[1].Bytes()),
		Receiver:    common.BytesToAddress(lg.Topics[2].Bytes()),
		Owner:       common.BytesToAddress(lg.Topics[3].Bytes()),
		Assets:      values[0].(*big.Int),
		Shares:      values[1].(*big.Int),
	}, nil
}

func decodeReward(c *chain.Client, lg types.Log) (db.RewardEvent, error) {
	if len(lg.Topics) < 2 {
		return db.RewardEvent{}, fmt.Errorf("reward: too few topics")
	}
	values, err := c.ABI.Events["RewardsDistributed"].Inputs.NonIndexed().Unpack(lg.Data)
	if err != nil {
		return db.RewardEvent{}, err
	}
	return db.RewardEvent{
		TxHash:           lg.TxHash,
		LogIndex:         lg.Index,
		BlockNumber:      lg.BlockNumber,
		BlockHash:        lg.BlockHash,
		Operator:         common.BytesToAddress(lg.Topics[1].Bytes()),
		Amount:           values[0].(*big.Int),
		TotalAssetsAfter: values[1].(*big.Int),
	}, nil
}
