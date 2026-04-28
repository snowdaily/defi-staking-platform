# Security Considerations

Living document. Updated as the codebase evolves.

## Threat Model

### Actors
- **User**: deposits / withdraws, expects fair share price
- **Operator**: pushes rewards, holds OPERATOR role (multisig in prod)
- **Admin**: can pause, upgrade, change parameters (multisig + timelock in prod)
- **Attacker**: external, may deploy malicious contracts, can sandwich, can flash-loan

## Known Attack Vectors and Mitigations

### 1. First Depositor Inflation Attack (a.k.a. Donation Attack)
**Vector**: Attacker deposits 1 wei → gets 1 share. Donates large amount of underlying directly to vault. Subsequent depositors get 0 shares due to rounding.

**Mitigation**: OpenZeppelin ERC4626 `_decimalsOffset()` virtual shares — makes attack uneconomical by orders of magnitude.

### 2. Reentrancy
**Vector**: Malicious token / hook re-enters during deposit/withdraw to exploit intermediate state.

**Mitigation**:
- `ReentrancyGuard` on all state-changing external functions
- Checks-Effects-Interactions ordering
- Pull pattern for any user-claimable rewards

### 3. Reward Sandwich / Front-Running (NOT MITIGATED in v1)
**Vector**: Attacker observes a pending `distributeRewards(amount)` tx in the public mempool. They deposit in the same block (in front of the operator's tx), wait for the reward to land, then redeem in the next block. Because rewards land as a step-function increase in `totalAssets`, the attacker captures yield they did not earn for the full duration the funds were supposedly staked.

**Status**: **Known limitation of v1.** No mitigation is implemented in code. This must be addressed before any mainnet deployment.

**Mitigations to implement before production** (any of these, ranked by effectiveness):
1. **Reward streaming** (preferred — used by Synthetix StakingRewards, sFrxETH): operator calls `notifyRewardAmount(amount, duration)`; rewards accrue linearly across `duration` instead of landing in one block.
2. **Withdrawal cooldown / queue**: redemption requires waiting N blocks after deposit, breaking the deposit→reward→redeem cycle.
3. **Private mempool / Flashbots**: operator submits the reward tx through a private relay so the mempool front-run window is closed.
4. **Commit/reveal**: operator commits to reward amount one block before revealing it, so attackers cannot deposit reactively.

**emergencyWithdraw note**: because `emergencyWithdraw` deliberately bypasses pause, this attack cannot be stopped by a paused vault. Reward streaming (mitigation 1) is the only architectural fix that does not weaken the emergency exit.

### 4. Rounding Errors
**Vector**: Repeated deposit/withdraw of dust amounts to drain rounding.

**Mitigation**:
- Round in vault's favor (down on shares-out, up on assets-in)
- Foundry invariant test: `totalAssets >= sum(userBalances)` after any sequence

### 5. Upgrade Risk (UUPS)
**Vector**: Compromised admin upgrades to malicious implementation.

**Mitigation**:
- Multisig + timelock for upgrades
- Storage layout linter in CI (detect storage collisions)
- Mandatory unit test for each new implementation against old state

### 6. Oracle Manipulation
**Vector**: If APR or external price feeds are used in critical paths, attacker manipulates.

**Mitigation**:
- v1 has no oracle dependency (exchange rate is internal, deterministic from totalAssets/totalSupply)
- If added: Chainlink with staleness checks + TWAP fallback

### 7. Pause Abuse
**Vector**: Operator pauses indefinitely, locking user funds.

**Mitigation**:
- `emergencyWithdraw` always callable by user (bypasses pause)
- Pause has max duration enforced on-chain

## Pre-Deployment Checklist

- [ ] Slither: 0 high/medium severity
- [ ] Mythril: clean
- [ ] 100% line coverage
- [ ] Invariant tests run for ≥10k iterations
- [ ] Fuzz tests run for ≥10k iterations on math
- [ ] Mainnet fork tests against current Lido state
- [ ] Storage layout snapshot committed
- [ ] Gas snapshot committed
- [ ] All admin functions behind multisig
- [ ] Timelock on parameter changes
- [ ] Bug bounty program scoped (Immunefi tier definition)
- [ ] At least one external audit (or self-audit checklist completed)

## Known Limitations (v1)

- Single yield source (no diversification)
- No slashing protection (yield source is mocked, so n/a in v1)
- No insurance fund
