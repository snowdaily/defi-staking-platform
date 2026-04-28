# Liquid Staking Protocol

Production-grade liquid staking protocol with full on-chain + off-chain stack. Built as a portfolio project demonstrating end-to-end DeFi engineering: smart contracts, indexing, backend services, dApp frontend.

## What It Does

Users deposit an ERC-20 → receive a yield-bearing liquid staking token (`stTKN`). Underlying assets accumulate rewards (pushed by an operator). Holders can redeem at any time at the current exchange rate.

ERC-4626 compliant. Reward-bearing model (exchange rate increases over time, no rebasing). Inflation-attack resistant via OpenZeppelin's virtual shares offset.

## Stack

| Layer | Tech |
|-------|------|
| Contracts | Solidity 0.8.24, Foundry, OpenZeppelin v5 |
| Indexer | Go 1.21+, PostgreSQL, go-ethereum |
| API | Go (REST via chi) |
| Reward Bot | Go (cron, EIP-1559 gas strategy, eth_call simulation) |
| Subgraph | The Graph (parallel implementation) |
| Frontend | Next.js 14, TypeScript, wagmi v2, viem, RainbowKit, Tailwind |
| Devnet | Anvil, Docker Compose |
| CI | GitHub Actions (forge build/test/coverage, Slither, go vet/test, pnpm lint/typecheck/build) |
| Monitoring | Prometheus, Grafana |

## What's In Each Package

```
contracts/    Solidity contracts + Foundry tests
  src/        StakingVault.sol, RewardDistributor.sol
  test/       49 tests: unit, fuzz (5 properties × 256 runs), invariant (3 × 128k calls)
  script/     Deploy.s.sol — full local + testnet deploy
backend/      Go monorepo (go 1.21)
  cmd/indexer/      WS event subscription, reorg-aware ingestion
  cmd/api/          REST: /tvl, /apr, /users/:addr/{position,history}, /rewards/recent
  cmd/rewardbot/    Cron-based reward push with simulation, gas cap, dry-run
  internal/chain/   ethclient wrapper + EIP-1559 TxSender + Signer abstraction
  internal/db/      pgx pool + embedded migrations + typed queries
  migrations/       SQL schema (events, indexer state, block trail, rate snapshots)
frontend/     Next.js 14 dApp
  app/        Stake / unstake / position / history
  lib/        wagmi config, ABIs, API client, formatting helpers
  components/ StatsHeader, StakeCard, PositionCard, HistoryList
subgraph/     The Graph subgraph (schema + AssemblyScript mappings)
docs/
  architecture.md       System overview + design decisions
  security.md           Threat model + attack vectors + mitigations
  audit-checklist.md    Pre-deploy gate
  gas-report.md         Gas costs + optimisation notes
  local-dev.md          End-to-end local setup
```

## Status

✅ Phase 1 — Smart contracts (49 tests pass, 100% line/function coverage, fuzz + invariant)
✅ Phase 2 — Indexer + DB schema
✅ Phase 3 — REST API
✅ Phase 4 — Reward bot
✅ Phase 5 — dApp
✅ Phase 6 — Subgraph + docs

## Quick Start

See [docs/local-dev.md](./docs/local-dev.md) for the full walkthrough. TL;DR:

```bash
make bootstrap   # install Foundry deps, Go deps, frontend deps
make up          # start Postgres + Anvil
cd contracts && forge script script/Deploy.s.sol --rpc-url http://localhost:8545 --broadcast
# In separate terminals:
cd backend && go run ./cmd/indexer
cd backend && go run ./cmd/api
cd frontend && pnpm dev
```

## Why This Project

DeFi engineering demands three things at once:
1. **Smart contract security** — Solidity, OpenZeppelin patterns, fuzz/invariant testing, attack-vector awareness
2. **Backend integration** — event indexing with reorg handling, signing, gas strategy, observability
3. **Full-stack delivery** — dApp, API, infra, CI

This repo is a single artifact that exercises all three at production quality.

## License

MIT
