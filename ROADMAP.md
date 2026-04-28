# Roadmap

Build order is designed so each phase produces a demoable artifact and follows strict TDD (test first, implementation second).

## Phase 1 — Smart Contracts MVP (Week 1–2)

**Goal:** Working ERC-4626 staking vault on a local Anvil node, 100% test coverage.

- [ ] Foundry project init, OpenZeppelin v5 dependency, CI lint
- [ ] `StakingVault.sol` (ERC-4626 compliant)
  - Tests first: deposit / withdraw / share-price math / edge cases (zero, dust, first depositor attack)
- [ ] `LiquidStakingToken.sol` (the share token)
- [ ] `RewardDistributor.sol` (operator pushes rewards → vault exchange rate increases)
  - Tests: reward accounting, late-deposit fairness, rounding direction
- [ ] `Treasury.sol` (UUPS upgradeable, fee collection)
- [ ] Access control (OZ AccessControl roles: ADMIN, OPERATOR, EMERGENCY)
- [ ] Pausable + emergency withdrawal path
- [ ] Foundry invariant tests (totalAssets >= totalSupply * minRate)
- [ ] Foundry fuzz tests on math functions
- [ ] Slither + Mythril in CI
- [ ] Gas snapshot in CI
- [ ] Deploy script for Anvil, Sepolia, Base Sepolia

## Phase 2 — Indexer + DB (Week 3)

**Goal:** Off-chain mirror of contract state, queryable by API.

- [ ] Go module init (Go 1.21+)
- [ ] PostgreSQL schema (users, deposits, withdrawals, reward_epochs, exchange_rate_history)
- [ ] `cmd/indexer`: subscribe to logs via WebSocket
- [ ] Reorg handling (track block hashes, rewind on detection)
- [ ] Retry + backoff for RPC failures
- [ ] Prometheus metrics (blocks behind, last event timestamp, RPC error rate)
- [ ] Integration test: spin up Anvil + indexer + Postgres, deposit on-chain, assert DB state

## Phase 3 — REST + GraphQL API (Week 3–4)

**Goal:** Frontend can query TVL, APR, user position, history.

- [ ] `cmd/api`: REST endpoints (`/tvl`, `/apr`, `/users/:addr/position`, `/users/:addr/history`)
- [ ] GraphQL schema via gqlgen
- [ ] APR calculation (rolling 7-day from exchange rate history)
- [ ] Caching (in-memory LRU for hot queries)
- [ ] OpenAPI spec auto-generation
- [ ] Integration tests against testcontainers Postgres

## Phase 4 — Reward Bot (Week 4)

**Goal:** Automated, safe, gas-aware reward distribution.

- [ ] `cmd/rewardbot`: cron-based reward submission
- [ ] Nonce manager (handles stuck tx, replacement)
- [ ] Gas strategy (EIP-1559, max fee cap)
- [ ] KMS-style key abstraction (env-based for dev, AWS KMS / HSM-ready interface)
- [ ] Dry-run mode (simulate via `eth_call` before sending)
- [ ] Alerting on failure (webhook)

## Phase 5 — Frontend dApp (Week 5)

**Goal:** Live demo on Sepolia + Vercel.

- [ ] Next.js 14 App Router init
- [ ] wagmi + viem + RainbowKit setup
- [ ] Stake / Unstake / Claim flows
- [ ] Live TVL, APR, user position from API
- [ ] Tx simulation before send (Tenderly API)
- [ ] Error states, loading states, optimistic UI
- [ ] Mobile responsive
- [ ] Vercel deploy

## Phase 6 — Subgraph + Hardening (Week 6)

**Goal:** Parallel indexer (The Graph) + production readiness.

- [ ] Subgraph schema + mappings
- [ ] Hosted Service / decentralized network deploy
- [ ] Mainnet fork tests (Foundry, fork from real Lido state)
- [ ] Audit checklist document (`docs/audit-checklist.md`)
- [ ] Attack vector analysis (`docs/security.md`): reentrancy, donation/inflation attack, oracle manipulation, sandwich
- [ ] Gas optimization report (before/after)
- [ ] Demo video + screenshots in README

## Out of Scope (v1)

- Actual validator delegation (would need real ETH staking integration). Mock yield source instead.
- Cross-chain bridging.
- Permissionless governance / DAO.

These may become v2 if v1 lands cleanly.

## Definition of Done (per phase)

1. All tests written before implementation. Red → green → refactor.
2. CI green (lint, test, coverage threshold).
3. Code review pass (via `superpowers:requesting-code-review`).
4. Demoable artifact (deployed contract / running service / live URL).
5. Documentation updated.
