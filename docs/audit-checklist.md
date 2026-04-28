# Audit Checklist

Self-audit gate before any external review. Every item must be checked off and evidence linked in PR description.

## Build & Tooling
- [ ] `forge build` clean (no warnings beyond known whitelist)
- [ ] `forge test -vvv` — all tests pass
- [ ] `forge coverage` — ≥95% lines, ≥90% branches
- [ ] `forge snapshot --check` — no unexpected gas regressions
- [ ] `slither .` — 0 high/medium severity findings
- [ ] `mythril analyze src/StakingVault.sol --execution-timeout 300` — no high severity
- [ ] `forge fmt --check` — formatted

## Solidity-Level Review
- [ ] No `tx.origin` for authorisation
- [ ] No `block.timestamp` used for randomness
- [ ] All external calls follow Checks-Effects-Interactions
- [ ] All state-changing externals have `nonReentrant` where applicable
- [ ] All `transfer` / `transferFrom` use SafeERC20
- [ ] No unchecked arithmetic in user-input paths
- [ ] No `selfdestruct`, no `delegatecall` to user-controlled targets
- [ ] All admin-only functions guarded by AccessControl
- [ ] Pausable applied to all entry points except emergency exit

## ERC-4626 Conformance
- [ ] `previewDeposit` matches `deposit` (fuzz test passes)
- [ ] `previewMint` matches `mint`
- [ ] `previewWithdraw` matches `withdraw`
- [ ] `previewRedeem` matches `redeem`
- [ ] `convertToAssets ∘ convertToShares` never inflates value
- [ ] `maxDeposit`, `maxMint`, `maxWithdraw`, `maxRedeem` correct under pause

## Economic Properties
- [ ] First-depositor inflation attack: victim still receives ≥99% of deposit value
- [ ] Reward distribution monotonically increases share price
- [ ] No path lets a user withdraw more than they deposited (excluding earned rewards)
- [ ] Rounding always favours the vault, never the user

## Upgradeability (UUPS — when upgrade is added in v2)
- [ ] Storage layout snapshot committed
- [ ] Storage layout linter (`forge inspect StorageLayout`) compared against previous version
- [ ] `_authorizeUpgrade` restricted to admin
- [ ] Init function has `initializer` modifier and is non-callable post-deploy

## Operational
- [ ] All admin functions documented with intended caller
- [ ] Multisig + timelock configuration document
- [ ] Pause max-duration enforced
- [ ] Emergency contact + escalation runbook
- [ ] Incident response: how to drain to safe vault under attack

## Pre-Mainnet
- [ ] Mainnet fork test against current target chain state
- [ ] Bug bounty program scoped (Immunefi)
- [ ] At least one external audit OR full ToB-style self-review
- [ ] Deployer key destruction plan
