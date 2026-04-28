"use client";

import { useAccount, useReadContract } from "wagmi";
import { vaultAbi, erc20Abi, VAULT_ADDRESS, ASSET_ADDRESS } from "@/lib/contracts";
import { fmt } from "@/lib/format";

export function PositionCard() {
  const { address } = useAccount();

  const { data: shares } = useReadContract({
    address: VAULT_ADDRESS,
    abi: vaultAbi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    query: { enabled: !!address, refetchInterval: 10_000 },
  });
  const { data: assetsForShares } = useReadContract({
    address: VAULT_ADDRESS,
    abi: vaultAbi,
    functionName: "convertToAssets",
    args: shares !== undefined ? [shares] : undefined,
    query: { enabled: shares !== undefined && shares > 0n, refetchInterval: 10_000 },
  });
  const { data: vaultDecimals } = useReadContract({
    address: VAULT_ADDRESS,
    abi: vaultAbi,
    functionName: "decimals",
  });
  const { data: assetDecimals } = useReadContract({
    address: ASSET_ADDRESS,
    abi: erc20Abi,
    functionName: "decimals",
  });
  const { data: assetSymbol } = useReadContract({
    address: ASSET_ADDRESS,
    abi: erc20Abi,
    functionName: "symbol",
  });

  if (!address) {
    return <div className="panel text-white/50">Connect a wallet to see your position.</div>;
  }

  const sym = assetSymbol ?? "";
  return (
    <div className="panel">
      <div className="text-xs uppercase tracking-wider text-white/50">Your position</div>
      <div className="mt-3 space-y-2">
        <Row label={`Shares (st${sym})`} value={fmt(shares, Number(vaultDecimals ?? 24), 4)} />
        <Row
          label="Underlying value"
          value={fmt(assetsForShares as bigint | undefined, Number(assetDecimals ?? 18), 4) + " " + sym}
        />
      </div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between font-mono">
      <span className="text-white/50">{label}</span>
      <span>{value}</span>
    </div>
  );
}
