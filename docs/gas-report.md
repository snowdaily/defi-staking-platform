# Gas Report

Snapshot of gas costs for the main user flows. Generated with `forge test --gas-report` and `forge snapshot`.

To regenerate:
```bash
cd contracts
forge snapshot
forge test --gas-report > docs/gas-report-raw.txt
```

## Vault — User Operations

| Operation | Approx Gas | Notes |
|-----------|-----------:|-------|
| `deposit(amount, receiver)` first time | ~108k | Includes ERC-20 transferFrom + mint + share calc |
| `deposit` subsequent | ~95k | Storage already warm |
| `withdraw` | ~100k | Burn shares + safeTransfer |
| `redeem` | ~94k | Same as withdraw |
| `emergencyWithdraw` | ~80k | Bypasses pause check, simpler path |

## Vault — Operator Operations

| Operation | Approx Gas |
|-----------|-----------:|
| `distributeRewards(amount)` | ~75k |
| `pause()` | ~30k |

## Distributor

| Operation | Approx Gas |
|-----------|-----------:|
| `fund(amount)` | ~50k |
| `distribute(amount)` | ~95k (includes vault distributeRewards inner call) |
| `distributeAll()` | ~95k |

## Optimisation Notes

Already applied:
- `optimizer_runs = 200` (balanced for deploy + runtime)
- Custom errors (cheaper than `require` strings)
- `immutable` for vault address in distributor
- Pre-approved vault in distributor constructor (saves an approve per cycle)

Possible future wins:
- Pack timestamp + counters into a single slot in any reward-rate contract added later
- Batch reward distribution if frequency is high enough to benefit from amortised overhead
- Use transient storage (EIP-1153) for reentrancy guard once Cancun is the floor
