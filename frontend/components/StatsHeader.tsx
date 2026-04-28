"use client";

import { useQuery } from "@tanstack/react-query";
import { useReadContract } from "wagmi";
import { api } from "@/lib/api";
import { fmt } from "@/lib/format";
import { erc20Abi, ASSET_ADDRESS } from "@/lib/contracts";

export function StatsHeader() {
  const tvl = useQuery({ queryKey: ["tvl"], queryFn: api.tvl, refetchInterval: 15_000 });
  const apr = useQuery({ queryKey: ["apr"], queryFn: api.apr, refetchInterval: 60_000 });

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

  // Robust BigInt parse — API may transiently return malformed strings.
  let tvlAmount: bigint | undefined;
  if (tvl.data) {
    try {
      tvlAmount = BigInt(tvl.data.totalAssets);
    } catch {
      tvlAmount = undefined;
    }
  }
  const aprPct = apr.data?.aprPct ?? null;
  const decimals = Number(assetDecimals ?? 18);

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      <Stat
        label="TVL"
        value={tvlAmount === undefined ? "—" : fmt(tvlAmount, decimals, 2)}
        unit={assetSymbol ?? ""}
      />
      <Stat label="APR (7d)" value={aprPct === null ? "—" : `${aprPct.toFixed(2)}%`} />
      <Stat label="Window" value={apr.data ? `${apr.data.windowDays.toFixed(1)} days` : "—"} />
    </div>
  );
}

function Stat({ label, value, unit }: { label: string; value: string; unit?: string }) {
  return (
    <div className="panel">
      <div className="text-xs uppercase tracking-wider text-white/50">{label}</div>
      <div className="mt-2 text-3xl font-mono">
        {value}
        {unit && <span className="text-base text-white/40 ml-1">{unit}</span>}
      </div>
    </div>
  );
}
