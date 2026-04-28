package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client wraps an ethclient with the parsed Vault ABI and helpers used by
// both the indexer and reward bot.
type Client struct {
	Eth   *ethclient.Client
	ABI   abi.ABI
	Vault common.Address
}

func NewClient(ctx context.Context, rpcURL string, vault common.Address) (*Client, error) {
	c, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	parsed, err := abi.JSON(strings.NewReader(VaultABI))
	if err != nil {
		return nil, fmt.Errorf("parse abi: %w", err)
	}
	return &Client{Eth: c, ABI: parsed, Vault: vault}, nil
}

// EventTopics is a stable list of the topic hashes the indexer subscribes to.
func (c *Client) EventTopics() []common.Hash {
	return []common.Hash{
		c.ABI.Events["Deposit"].ID,
		c.ABI.Events["Withdraw"].ID,
		c.ABI.Events["RewardsDistributed"].ID,
	}
}

// FilterQuery returns the geth FilterQuery for the configured Vault.
func (c *Client) FilterQuery(from, to *big.Int) ethereum.FilterQuery {
	return ethereum.FilterQuery{
		FromBlock: from,
		ToBlock:   to,
		Addresses: []common.Address{c.Vault},
		Topics:    [][]common.Hash{c.EventTopics()},
	}
}

// HeaderAt returns the block header at `n`.
func (c *Client) HeaderAt(ctx context.Context, n uint64) (*types.Header, error) {
	return c.Eth.HeaderByNumber(ctx, new(big.Int).SetUint64(n))
}

// TotalAssets reads vault.totalAssets() at the latest block.
func (c *Client) TotalAssets(ctx context.Context) (*big.Int, error) {
	return c.callBigInt(ctx, "totalAssets")
}

// TotalSupply reads vault.totalSupply() at the latest block.
func (c *Client) TotalSupply(ctx context.Context) (*big.Int, error) {
	return c.callBigInt(ctx, "totalSupply")
}

func (c *Client) callBigInt(ctx context.Context, method string) (*big.Int, error) {
	data, err := c.ABI.Pack(method)
	if err != nil {
		return nil, err
	}
	out, err := c.Eth.CallContract(ctx, ethereum.CallMsg{To: &c.Vault, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.ABI.Unpack(method, out)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("%s returned empty", method)
	}
	v, ok := res[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("%s: unexpected type %T", method, res[0])
	}
	return v, nil
}
