# Subgraph

Parallel indexer using The Graph. Implemented alongside the Go indexer to demonstrate familiarity with both stacks.

## Setup

```bash
pnpm install
pnpm codegen
pnpm build
```

## Deploy

1. Get a deploy key from <https://thegraph.com/studio>.
2. Update the `address` and `startBlock` in `subgraph.yaml` to match your deployed vault.
3. Copy the contract ABI:
   ```bash
   mkdir -p abis && cp ../contracts/out/StakingVault.sol/StakingVault.json abis/StakingVault.json
   ```
4. Authenticate and deploy:
   ```bash
   graph auth --studio <DEPLOY_KEY>
   pnpm deploy
   ```

## Sample query

```graphql
{
  vaultMetric(id: "0x01") {
    totalDeposited
    totalWithdrawn
    totalRewards
    lastTotalAssets
    updatedAt
  }
  users(first: 10, orderBy: totalDeposited, orderDirection: desc) {
    id
    totalDeposited
    totalWithdrawn
    depositCount
  }
}
```
