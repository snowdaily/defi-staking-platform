"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { fmt } from "@/lib/format";

export function StatsHeader() {
  const tvl = useQuery({ queryKey: ["tvl"], queryFn: api.tvl, refetchInterval: 15_000 });
  const apr = useQuery({ queryKey: ["apr"], queryFn: api.apr, refetchInterval: 60_000 });

  const tvlAmount = tvl.data ? BigInt(tvl.data.totalAssets) : undefined;
  const aprPct = apr.data?.aprPct ?? null;

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      <Stat label="TVL" value={tvlAmount === undefined ? "—" : fmt(tvlAmount, 18, 2)} unit="MOCK" />
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
