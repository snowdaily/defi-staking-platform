package chain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Signer abstracts transaction signing.
//
// Two implementations: EnvSigner (private key from env) for dev, and a
// production-ready interface for KMS / HSM in the future. Reward bot only
// depends on this interface, so swapping the backend doesn't change call sites.
type Signer interface {
	Address() common.Address
	SignTx(ctx context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
}

type EnvSigner struct {
	priv *ecdsa.PrivateKey
	addr common.Address
}

func NewEnvSigner(hexKey string) (*EnvSigner, error) {
	if hexKey == "" {
		return nil, errors.New("empty private key")
	}
	hexKey = strings.TrimPrefix(hexKey, "0x")
	priv, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return nil, err
	}
	return &EnvSigner{priv: priv, addr: crypto.PubkeyToAddress(priv.PublicKey)}, nil
}

// DryRunSigner satisfies Signer without holding any key material. It reports
// a deterministic placeholder address and refuses to sign — used by the
// reward bot's dry-run mode where transactions are only simulated.
type DryRunSigner struct{}

func (DryRunSigner) Address() common.Address { return common.Address{} }

func (DryRunSigner) SignTx(_ context.Context, _ *types.Transaction, _ *big.Int) (*types.Transaction, error) {
	return nil, errors.New("dry-run signer cannot sign transactions")
}

func (s *EnvSigner) Address() common.Address { return s.addr }

func (s *EnvSigner) SignTx(_ context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return types.SignTx(tx, types.LatestSignerForChainID(chainID), s.priv)
}
