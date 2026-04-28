# Architecture

## High-Level Flow

```
┌─────────┐    deposit    ┌──────────────┐
│  User   │ ────────────► │ StakingVault │
│ Wallet  │ ◄──────────── │  (ERC-4626)  │
└─────────┘    stTKN      └──────┬───────┘
                                  │
                                  │ holds underlying
                                  ▼
                         ┌──────────────────┐
                         │ Yield Source     │
                         │ (mock in v1)     │
                         └──────────────────┘
                                  ▲
                                  │ pushes rewards
                         ┌────────┴─────────┐
                         │ RewardBot (Go)   │
                         │ scheduled cron   │
                         └──────────────────┘

┌──────────────┐  events  ┌─────────────┐
│ StakingVault │ ───────► │ Indexer     │ ──► PostgreSQL
└──────────────┘   logs   │ (Go)        │
                          └─────────────┘
                                                  ▲
                                                  │ queries
┌─────────────┐    HTTP/GraphQL    ┌──────────────┴──┐
│  Next.js    │ ─────────────────► │  API (Go)       │
│  dApp       │ ◄───────────────── │  REST + GraphQL │
└─────────────┘                    └─────────────────┘
       │
       │ wagmi/viem (read on-chain state directly)
       ▼
   ┌──────────┐
   │   RPC    │
   └──────────┘
```

## Why Both Indexer and Direct RPC

- **Direct RPC (frontend)**: live, trustless reads (user balance, allowances, current exchange rate)
- **Indexer + API**: aggregated and historical data (TVL over time, APR, user history) — too expensive to compute via RPC

## Why Both Go Indexer and Subgraph

Demonstrates two real-world approaches:

| Aspect | Custom Go Indexer | The Graph |
|--------|-------------------|-----------|
| Control | Full | Limited |
| Reorg handling | Manual | Built-in |
| Hosting | Self-hosted | Decentralized / Hosted |
| Query language | SQL / GraphQL | GraphQL only |
| Latency | Lower (single hop) | Higher |
| When to choose | Complex aggregations, custom logic | Simple read patterns, easy ops |

Production protocols often run both: subgraph for public dApp queries, custom indexer for internal analytics.

## Key Design Decisions

### Reward-bearing vs Rebasing

Choosing **reward-bearing** (exchange rate increases over time, like rETH/cbETH) over rebasing (balance changes, like stETH).

Reasons:
- Cleaner DeFi composability (no balance changes mid-tx)
- ERC-4626 native fit
- Simpler accounting downstream (DEX pools, lending markets)

### UUPS over Transparent Proxy

Smaller deployment cost, upgrade logic in implementation (more flexible).

### First-Depositor Inflation Attack Mitigation

Use OpenZeppelin's "virtual shares" approach (decimals offset) — well-tested defense against the donation attack.

### Reorg Strategy (Indexer)

Track last N block hashes (default N=64 ≈ 12min on Ethereum). On each new block, verify parent hash matches stored hash. On mismatch, walk back until match found, delete dependent rows, re-process.
