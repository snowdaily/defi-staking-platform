# Liquid Staking Protocol

[![Contracts CI](https://github.com/snowdaily/defi-staking-platform/actions/workflows/contracts.yml/badge.svg)](https://github.com/snowdaily/defi-staking-platform/actions/workflows/contracts.yml)
[![Backend CI](https://github.com/snowdaily/defi-staking-platform/actions/workflows/backend.yml/badge.svg)](https://github.com/snowdaily/defi-staking-platform/actions/workflows/backend.yml)
[![Frontend CI](https://github.com/snowdaily/defi-staking-platform/actions/workflows/frontend.yml/badge.svg)](https://github.com/snowdaily/defi-staking-platform/actions/workflows/frontend.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Solidity](https://img.shields.io/badge/Solidity-0.8.24-363636?logo=solidity)](https://soliditylang.org/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen.svg)](#testing)

Portfolio-quality liquid staking protocol — full on-chain + off-chain stack in a single repo. Implements **production engineering practices** (ERC-4626, fuzz + invariant testing, reorg-aware indexing, signer abstraction, full CI) but is **not audited and not production-deployed**.

> **Disclaimer.** This is a portfolio / educational project. The contracts have **not** been externally audited and the protocol has **not** been deployed to mainnet. Do not deploy with real funds without a professional audit, bug bounty program, multisig admin, reward-streaming mitigation for the front-running vector documented in [docs/security.md](./docs/security.md), and the rest of [docs/audit-checklist.md](./docs/audit-checklist.md).

---

## What It Does

Users deposit an ERC-20 and receive a yield-bearing liquid staking token (`stTKN`). The vault accumulates rewards (pushed by an operator), and holders can redeem at the current exchange rate at any time.

- **ERC-4626 compliant** vault interface
- **Reward-bearing** model — exchange rate goes up over time (no rebasing, simpler DeFi composability)
- **Inflation-attack resistant** via OpenZeppelin virtual-shares offset
- **Pausable** with always-callable `emergencyWithdraw` so admin pause cannot lock user funds

## Architecture

```
┌─────────┐  deposit/redeem   ┌──────────────┐
│  User   │ ─────────────────►│ StakingVault │  (ERC-4626 + AccessControl + Pausable)
│ Wallet  │ ◄──────────────── │  + stTKN     │
└─────────┘     stTKN         └──────┬───────┘
                                     │ events
                              ┌──────▼─────────┐
                              │ Go Indexer     │──► PostgreSQL
                              │ (reorg-aware)  │
                              └──────┬─────────┘
                                     │
┌─────────┐    HTTP/JSON      ┌──────▼─────────┐
│ Next.js │ ─────────────────►│  Go API        │
│  dApp   │◄──────────────────│  REST          │
└─────────┘                   └────────────────┘

┌─────────────────┐  scheduled  ┌──────────────┐
│ Go Reward Bot   │ ───────────►│ StakingVault │
│ cron + EIP-1559 │             └──────────────┘
└─────────────────┘
```

## Tech Stack

| Layer | Stack |
|-------|-------|
| **Smart Contracts** | Solidity 0.8.24 · Foundry · OpenZeppelin v5 |
| **Indexer** | Go 1.21 · go-ethereum · PostgreSQL · pgx v5 |
| **API** | Go · chi · Prometheus · zerolog |
| **Reward Bot** | Go · robfig/cron · EIP-1559 dynamic fee · eth_call simulation · KMS-ready signer abstraction |
| **Subgraph** | The Graph · AssemblyScript |
| **Frontend** | Next.js 14 (App Router) · TypeScript · wagmi v2 · viem · RainbowKit · Tailwind |
| **Devnet** | Anvil · Docker Compose |
| **CI** | GitHub Actions · Slither static analysis · race-detector tests |
| **Monitoring** | Prometheus · Grafana |

## Highlights

**Smart contracts**
- 49 tests, **100 % line and function coverage**
- 5 fuzz properties at 256 runs each, 3 invariants at ~128 k calls each
- Slither clean (medium+ findings), forge fmt enforced in CI
- Custom errors throughout (gas-friendly), gas snapshot committed

**Indexer**
- WebSocket-style log filtering, batched range processing (1 000 blocks/iteration)
- 64-block hash trail with automatic rewind on reorg detection
- Embedded SQL migrations — `go run` self-applies on startup
- Prometheus metrics: `blocks_behind`, `events_processed_total{event}`, `reorgs_total`

**Reward bot**
- Pre-flight `eth_call` simulation — fail fast before paying gas
- EIP-1559 dynamic fee (`baseFee × 2 + tip`) with hard cap
- Dry-run mode for staging, signer abstraction (env-key today, KMS/HSM tomorrow)
- Cron-driven; one configurable schedule expression

**dApp**
- One-click stake / unstake (auto-detect approval), live position card, history table
- Multi-chain config (Anvil / Sepolia / Base Sepolia / Mainnet)
- Tx hash + mining state inline; Max balance one-tap

## Repository Layout

```
contracts/    Solidity + Foundry  (StakingVault, RewardDistributor, deploy script)
backend/      Go monorepo         (cmd/indexer, cmd/api, cmd/rewardbot)
frontend/     Next.js 14 dApp     (app router + wagmi + RainbowKit)
subgraph/     The Graph subgraph  (schema + AssemblyScript mappings)
docs/         Architecture, security, audit checklist, gas report, local-dev
```

## Quick Start

Full walkthrough in [docs/local-dev.md](./docs/local-dev.md). TL;DR:

```bash
make bootstrap                            # install Foundry, Go, frontend deps
make up                                   # Postgres + Anvil via docker compose
cd contracts && forge script script/Deploy.s.sol \
  --rpc-url http://localhost:8545 --broadcast

# In separate terminals
cd backend && go run ./cmd/indexer
cd backend && go run ./cmd/api
cd frontend && pnpm dev                   # http://localhost:3000
```

## Testing

```bash
cd contracts
forge test                                # 49 tests
forge coverage --no-match-coverage 'test|mocks'   # 100 % lines / 100 % funcs
forge test --match-path 'test/*.fuzz.t.sol'       # 5 fuzz × 256 runs
forge test --match-path 'test/*.invariant.t.sol'  # 3 invariants × ~128 k calls
slither contracts                         # clean for medium+
```

## Documentation

- [Architecture & design decisions](./docs/architecture.md)
- [Threat model & security](./docs/security.md)
- [Pre-deployment audit checklist](./docs/audit-checklist.md)
- [Gas report](./docs/gas-report.md)
- [Local development walkthrough](./docs/local-dev.md)

## Why This Project

DeFi engineering demands three things at once:

1. **Smart contract security** — Solidity discipline, OpenZeppelin patterns, fuzz / invariant testing, attack-vector awareness
2. **Backend integration** — event indexing with reorg handling, transaction signing, gas strategy, observability
3. **Full-stack delivery** — dApp, API, infrastructure, CI / CD

This repo exercises all three with production engineering patterns, in a single reproducible artifact.

## What This Repo Is — And Is Not

**Is**: a working ERC-4626 vault + indexer + API + reward bot + dApp with reproducible CI; an honest demonstration of how each layer is built; a starting point another engineer can read end-to-end in an afternoon.

**Is not**: a deployable product. The gap between "production engineering practices" and "production-deployed" is real, and this repo does not try to hide it. The known gaps before mainnet are:

| Gap | Severity | Where it's documented |
|-----|----------|-----------------------|
| Reward sandwich / front-running of `distributeRewards` | High | [docs/security.md §3](./docs/security.md) |
| No external audit / bug bounty | Blocker | [docs/audit-checklist.md](./docs/audit-checklist.md) |
| Single admin/operator key (should be multisig + timelock) | High | `script/Deploy.s.sol` env-driven, but no Safe wiring |
| `MockERC20` is open-mint — testing only | Blocker | `test/mocks/MockERC20.sol` |
| Reward bot has no stuck-tx replacement / nonce manager | Medium | `internal/chain/sender.go` |
| Mock yield source — no real validator delegation | Architectural | This is v1 by design |
| No HA on Postgres / RPC, no oncall, no SOC2 | Operational | Out of scope for v1 |

The cost of going from here to "$1M TVL on mainnet" is roughly: 2 independent audits ($150k–300k), reward-streaming refactor (2 weeks), multisig + timelock wiring (1 week), real yield-source integration (4–8 weeks), HA infra + monitoring + oncall (4 weeks), Immunefi bounty, and legal review.

Knowing what's missing is part of the work this repo demonstrates.

## License

[MIT](./LICENSE) — free to use, fork, modify, redistribute, and relicense, with attribution.
