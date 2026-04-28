"use client";

import { useQuery } from "@tanstack/react-query";
import { useAccount } from "wagmi";
import { api } from "@/lib/api";
import { fmt } from "@/lib/format";

export function HistoryList() {
  const { address } = useAccount();
  const { data, isLoading, error } = useQuery({
    queryKey: ["history", address],
    queryFn: () => api.history(address!),
    enabled: !!address,
    refetchInterval: 30_000,
  });

  if (!address) return null;

  return (
    <div className="panel">
      <div className="text-xs uppercase tracking-wider text-white/50 mb-3">Recent activity</div>
      {isLoading && <div className="text-white/40 text-sm">Loading…</div>}
      {error && <div className="text-err text-sm">Failed to load history</div>}
      {data && data.length === 0 && <div className="text-white/40 text-sm">No activity yet</div>}
      {data && data.length > 0 && (
        <table className="w-full text-sm font-mono">
          <thead className="text-white/40">
            <tr>
              <th className="text-left font-normal py-1">Type</th>
              <th className="text-right font-normal">Assets</th>
              <th className="text-right font-normal">Block</th>
              <th className="text-right font-normal">When</th>
            </tr>
          </thead>
          <tbody>
            {data.map((e) => (
              <tr key={`${e.kind}-${e.blockNumber}-${e.shares}`} className="border-t border-white/5">
                <td
                  className={`py-1 ${e.kind === "deposit" ? "text-ok" : "text-warn"}`}
                >
                  {e.kind}
                </td>
                <td className="text-right">{fmt(BigInt(e.assets), 18, 2)}</td>
                <td className="text-right">{e.blockNumber}</td>
                <td className="text-right text-white/50">
                  {new Date(e.timestamp).toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
