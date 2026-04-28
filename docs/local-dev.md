# Local Development Guide

End-to-end setup for running the entire stack on your laptop.

## Prerequisites

- Foundry (`curl -L https://foundry.paradigm.xyz | bash && foundryup`)
- Go 1.21+
- Node.js 20+ and pnpm
- Docker

## 1. Start infra

```bash
make up
```

This starts:
- PostgreSQL on `localhost:5432` (user/pass: `staking/staking`)
- Anvil on `localhost:8545` (chain id 31337)

## 2. Deploy contracts

```bash
cd contracts
forge script script/Deploy.s.sol \
  --rpc-url http://localhost:8545 \
  --broadcast
```

Note the printed `Asset`, `Vault`, and `Distributor` addresses.

## 3. Run the indexer

```bash
cd backend
export DATABASE_URL=postgres://staking:staking@localhost:5432/staking?sslmode=disable
export RPC_HTTP_URL=http://localhost:8545
export VAULT_ADDRESS=<vault-address-from-step-2>
go run ./cmd/indexer
```

The indexer auto-applies migrations on startup.

## 4. Run the API

```bash
go run ./cmd/api
# Listening on :8080 ; metrics on :9100
```

Try:
```bash
curl http://localhost:8080/api/v1/tvl
curl http://localhost:8080/api/v1/apr
```

## 5. Run the reward bot (dry-run)

```bash
DRY_RUN=true \
VAULT_ADDRESS=<vault> \
REWARD_SCHEDULE="*/30 * * * * *" \
go run ./cmd/rewardbot
```

For real signing on Anvil, use one of the prefunded keys:
```bash
OPERATOR_PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
VAULT_ADDRESS=<vault> \
go run ./cmd/rewardbot
```

## 6. Run the dApp

```bash
cd frontend
cp .env.example .env.local
# Edit .env.local with the addresses from step 2
pnpm install
pnpm dev
# Open http://localhost:3000
```

Connect via MetaMask:
- Add network: chain id 31337, RPC `http://localhost:8545`
- Import a prefunded Anvil key

## 7. Trigger some events

In a separate terminal:
```bash
# Mint test tokens to your wallet (use cast)
cast send <ASSET_ADDR> "mint(address,uint256)" <YOUR_ADDR> 1000ether \
  --rpc-url http://localhost:8545 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
```

Then stake / unstake from the UI. Watch:
- The indexer logs (events ingested)
- API responses (TVL updates)
- The dApp position card refresh
