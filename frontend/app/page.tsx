"use client";

import { ConnectButton } from "@rainbow-me/rainbowkit";
import { StatsHeader } from "@/components/StatsHeader";
import { StakeCard } from "@/components/StakeCard";
import { PositionCard } from "@/components/PositionCard";
import { HistoryList } from "@/components/HistoryList";

export default function Home() {
  return (
    <main className="min-h-screen">
      <header className="border-b border-white/5">
        <div className="max-w-5xl mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded bg-accent/20 border border-accent/30 grid place-items-center text-accent">
              ◇
            </div>
            <div className="font-semibold">Liquid Staking</div>
          </div>
          <ConnectButton showBalance={false} />
        </div>
      </header>

      <div className="max-w-5xl mx-auto px-4 py-8 space-y-6">
        <StatsHeader />
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <StakeCard />
          <PositionCard />
        </div>
        <HistoryList />
        <footer className="text-xs text-white/30 text-center pt-8">
          ERC-4626 staking vault · open source ·{" "}
          <a href="https://github.com/" className="hover:text-white/60">
            github
          </a>
        </footer>
      </div>
    </main>
  );
}
