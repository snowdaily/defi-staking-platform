// Formatting helpers used across the dApp UI.

import { formatUnits, parseUnits } from "viem";

export function fmt(amount: bigint | undefined, decimals: number, frac = 4): string {
  if (amount === undefined) return "—";
  const s = formatUnits(amount, decimals);
  const [int, dec = ""] = s.split(".");
  const trimmed = dec.padEnd(frac, "0").slice(0, frac);
  return frac > 0 ? `${int}.${trimmed}` : int;
}

export function tryParse(input: string, decimals: number): bigint | null {
  if (!input || !/^\d*\.?\d*$/.test(input)) return null;
  try {
    return parseUnits(input, decimals);
  } catch {
    return null;
  }
}

export function shortAddr(a?: string): string {
  if (!a) return "";
  return `${a.slice(0, 6)}…${a.slice(-4)}`;
}
