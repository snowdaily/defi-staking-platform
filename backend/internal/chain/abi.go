package chain

// Subset of the StakingVault ABI: just the events the indexer cares about,
// plus the read-only views used to snapshot the exchange rate.
//
// Hand-written to avoid the abigen toolchain dependency. If/when the contract
// surface grows, switch to `abigen` generation.
const VaultABI = `[
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true,  "name": "sender", "type": "address"},
      {"indexed": true,  "name": "owner",  "type": "address"},
      {"indexed": false, "name": "assets", "type": "uint256"},
      {"indexed": false, "name": "shares", "type": "uint256"}
    ],
    "name": "Deposit",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true,  "name": "sender",   "type": "address"},
      {"indexed": true,  "name": "receiver", "type": "address"},
      {"indexed": true,  "name": "owner",    "type": "address"},
      {"indexed": false, "name": "assets",   "type": "uint256"},
      {"indexed": false, "name": "shares",   "type": "uint256"}
    ],
    "name": "Withdraw",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true,  "name": "operator",         "type": "address"},
      {"indexed": false, "name": "amount",           "type": "uint256"},
      {"indexed": false, "name": "totalAssetsAfter", "type": "uint256"}
    ],
    "name": "RewardsDistributed",
    "type": "event"
  },
  {
    "inputs": [],
    "name": "totalAssets",
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "totalSupply",
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [{"name": "amount", "type": "uint256"}],
    "name": "distributeRewards",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]`
