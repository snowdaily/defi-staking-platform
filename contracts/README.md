# Contracts

Foundry workspace for the staking protocol contracts.

## Setup

Install Foundry:
```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

Install deps:
```bash
forge install
```

## Commands

```bash
forge build              # Compile
forge test               # Run all tests
forge test -vvv          # Verbose (show traces on failure)
forge coverage           # Coverage report
forge snapshot           # Gas snapshot
slither .                # Static analysis (requires `pip install slither-analyzer`)
```

## Layout

```
src/         Production contracts
test/        Foundry tests (.t.sol)
script/      Deployment scripts
```

## Standards

- Solidity ^0.8.24
- OpenZeppelin v5
- ERC-4626 for the vault
- UUPS for upgradeability
- 100% line coverage target
- Invariant + fuzz tests for all math
