import { BigInt, Bytes } from "@graphprotocol/graph-ts";
import {
  Deposit as DepositEvent,
  Withdraw as WithdrawEvent,
  RewardsDistributed as RewardsEvent,
} from "../generated/StakingVault/StakingVault";
import { Deposit, Withdrawal, RewardEvent, User, VaultMetric } from "../generated/schema";

const METRIC_KEY = Bytes.fromHexString("0x01") as Bytes;

function loadOrCreateUser(addr: Bytes): User {
  let u = User.load(addr);
  if (u == null) {
    u = new User(addr);
    u.totalDeposited = BigInt.zero();
    u.totalWithdrawn = BigInt.zero();
    u.depositCount = 0;
    u.withdrawCount = 0;
  }
  return u;
}

function loadOrCreateMetric(): VaultMetric {
  let m = VaultMetric.load(METRIC_KEY);
  if (m == null) {
    m = new VaultMetric(METRIC_KEY);
    m.totalDeposited = BigInt.zero();
    m.totalWithdrawn = BigInt.zero();
    m.totalRewards = BigInt.zero();
    m.lastTotalAssets = BigInt.zero();
    m.updatedAt = BigInt.zero();
  }
  return m;
}

export function handleDeposit(event: DepositEvent): void {
  const id = event.transaction.hash.concatI32(event.logIndex.toI32());
  const d = new Deposit(id);
  d.sender = event.params.sender;
  d.owner = event.params.owner;
  d.assets = event.params.assets;
  d.shares = event.params.shares;
  d.blockNumber = event.block.number;
  d.blockTimestamp = event.block.timestamp;
  d.txHash = event.transaction.hash;
  d.save();

  const u = loadOrCreateUser(event.params.owner);
  u.totalDeposited = u.totalDeposited.plus(event.params.assets);
  u.depositCount = u.depositCount + 1;
  u.save();

  const m = loadOrCreateMetric();
  m.totalDeposited = m.totalDeposited.plus(event.params.assets);
  m.updatedAt = event.block.timestamp;
  m.save();
}

export function handleWithdraw(event: WithdrawEvent): void {
  const id = event.transaction.hash.concatI32(event.logIndex.toI32());
  const w = new Withdrawal(id);
  w.sender = event.params.sender;
  w.receiver = event.params.receiver;
  w.owner = event.params.owner;
  w.assets = event.params.assets;
  w.shares = event.params.shares;
  w.blockNumber = event.block.number;
  w.blockTimestamp = event.block.timestamp;
  w.txHash = event.transaction.hash;
  w.save();

  const u = loadOrCreateUser(event.params.owner);
  u.totalWithdrawn = u.totalWithdrawn.plus(event.params.assets);
  u.withdrawCount = u.withdrawCount + 1;
  u.save();

  const m = loadOrCreateMetric();
  m.totalWithdrawn = m.totalWithdrawn.plus(event.params.assets);
  m.updatedAt = event.block.timestamp;
  m.save();
}

export function handleRewards(event: RewardsEvent): void {
  const id = event.transaction.hash.concatI32(event.logIndex.toI32());
  const r = new RewardEvent(id);
  r.operator = event.params.operator;
  r.amount = event.params.amount;
  r.totalAssetsAfter = event.params.totalAssetsAfter;
  r.blockNumber = event.block.number;
  r.blockTimestamp = event.block.timestamp;
  r.txHash = event.transaction.hash;
  r.save();

  const m = loadOrCreateMetric();
  m.totalRewards = m.totalRewards.plus(event.params.amount);
  m.lastTotalAssets = event.params.totalAssetsAfter;
  m.updatedAt = event.block.timestamp;
  m.save();
}
