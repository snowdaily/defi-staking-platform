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

### 3. Reward Manipulation / Front-Running
**Vector**: Attacker front-runs `distributeRewards()` to deposit just before, withdraws right after.

**Mitigation**:
- Cooldown / lock-up on withdrawals (configurable)
- Distribute rewards over a window (drip) instead of step function
- Document this as a known economic property if drip not used

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
