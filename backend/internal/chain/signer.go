package chain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"

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
	if hexKey[:2] == "0x" {
		hexKey = hexKey[2:]
	}
	priv, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return nil, err
	}
	return &EnvSigner{priv: priv, addr: crypto.PubkeyToAddress(priv.PublicKey)}, nil
}

func (s *EnvSigner) Address() common.Address { return s.addr }

func (s *EnvSigner) SignTx(_ context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return types.SignTx(tx, types.LatestSignerForChainID(chainID), s.priv)
}
