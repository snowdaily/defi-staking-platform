// Thin client over the Go API. Returns plain JSON; component layer handles loading + errors.

const BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

export interface TVLResponse {
  totalAssets: string;
  totalSupply: string;
}

export interface APRResponse {
  aprPct: number;
  windowDays: number;
  points: number;
}

export interface HistoryEntry {
  kind: "deposit" | "withdraw";
  blockNumber: number;
  timestamp: string;
  assets: string;
  shares: string;
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`, { cache: "no-store" });
  if (!res.ok) throw new Error(`API ${path}: ${res.status}`);
  return res.json();
}

export const api = {
  tvl: () => get<TVLResponse>("/tvl"),
  apr: () => get<APRResponse>("/apr"),
  history: (addr: string) => get<HistoryEntry[]>(`/users/${addr}/history`),
};
