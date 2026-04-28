package chain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

// TxSender combines a chain client and a Signer to build, simulate, and send
// EIP-1559 transactions safely. Handles nonce + gas concerns.
type TxSender struct {
	Client     *Client
	Signer     Signer
	ChainID    *big.Int
	MaxGasGwei int64
}

// SendCall builds an EIP-1559 transaction calling `to` with `data`, simulates it
// via eth_call, and (unless dryRun) signs and broadcasts it.
func (s *TxSender) SendCall(ctx context.Context, to common.Address, data []byte, dryRun bool) (common.Hash, error) {
	from := s.Signer.Address()

	// Simulate first — fail fast on revert.
	if _, err := s.Client.Eth.CallContract(ctx, ethereum.CallMsg{
		From: from, To: &to, Data: data,
	}, nil); err != nil {
		return common.Hash{}, fmt.Errorf("simulation reverted: %w", err)
	}

	gas, err := s.Client.Eth.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &to, Data: data})
	if err != nil {
		return common.Hash{}, fmt.Errorf("estimate gas: %w", err)
	}
	gas = gas * 12 / 10 // 20% buffer

	tip, err := s.Client.Eth.SuggestGasTipCap(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	head, err := s.Client.Eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return common.Hash{}, err
	}

	baseFee := head.BaseFee
	if baseFee == nil {
		baseFee = big.NewInt(0)
	}
	maxFee := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), tip)

	feeCap := new(big.Int).Mul(big.NewInt(s.MaxGasGwei), big.NewInt(1_000_000_000))
	if maxFee.Cmp(feeCap) > 0 {
		return common.Hash{}, fmt.Errorf("maxFee %s gwei exceeds cap %d", new(big.Int).Quo(maxFee, big.NewInt(1_000_000_000)).String(), s.MaxGasGwei)
	}

	if dryRun {
		log.Info().
			Uint64("gas", gas).
			Str("maxFeeGwei", new(big.Int).Quo(maxFee, big.NewInt(1_000_000_000)).String()).
			Msg("dry-run: skipping send")
		return common.Hash{}, nil
	}

	nonce, err := s.Client.Eth.PendingNonceAt(ctx, from)
	if err != nil {
		return common.Hash{}, err
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   s.ChainID,
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: maxFee,
		Gas:       gas,
		To:        &to,
		Value:     big.NewInt(0),
		Data:      data,
	})

	signed, err := s.Signer.SignTx(ctx, tx, s.ChainID)
	if err != nil {
		return common.Hash{}, err
	}
	if err := s.Client.Eth.SendTransaction(ctx, signed); err != nil {
		return common.Hash{}, err
	}
	log.Info().Str("tx", signed.Hash().Hex()).Uint64("nonce", nonce).Msg("tx sent")
	return signed.Hash(), nil
}

// WaitForReceipt polls until the tx is mined or the context times out.
// Distinguishes "not yet mined" (NotFound, retried) from real errors
// (RPC failure, auth, etc., returned immediately).
func (s *TxSender) WaitForReceipt(ctx context.Context, h common.Hash) (*types.Receipt, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		r, err := s.Client.Eth.TransactionReceipt(ctx, h)
		if err == nil {
			if r.Status == types.ReceiptStatusFailed {
				return r, errors.New("tx reverted")
			}
			return r, nil
		}
		if !errors.Is(err, ethereum.NotFound) {
			return nil, fmt.Errorf("receipt fetch: %w", err)
		}
		time.Sleep(time.Second)
	}
}
