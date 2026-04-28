"use client";

import { useState, useMemo } from "react";
import { useAccount, useReadContract, useWriteContract, useWaitForTransactionReceipt } from "wagmi";
import { erc20Abi, vaultAbi, ASSET_ADDRESS, VAULT_ADDRESS } from "@/lib/contracts";
import { fmt, tryParse } from "@/lib/format";

type Mode = "stake" | "unstake";

export function StakeCard() {
  const { address } = useAccount();
  const [mode, setMode] = useState<Mode>("stake");
  const [input, setInput] = useState("");

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
  const { data: vaultDecimals } = useReadContract({
    address: VAULT_ADDRESS,
    abi: vaultAbi,
    functionName: "decimals",
  });
  const { data: assetBalance, refetch: refetchAssetBal } = useReadContract({
    address: ASSET_ADDRESS,
    abi: erc20Abi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    query: { enabled: !!address },
  });
  const { data: shareBalance, refetch: refetchShareBal } = useReadContract({
    address: VAULT_ADDRESS,
    abi: vaultAbi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    query: { enabled: !!address },
  });
  const { data: allowance, refetch: refetchAllowance } = useReadContract({
    address: ASSET_ADDRESS,
    abi: erc20Abi,
    functionName: "allowance",
    args: address ? [address, VAULT_ADDRESS] : undefined,
    query: { enabled: !!address },
  });

  const inputDecimals = mode === "stake" ? Number(assetDecimals ?? 18) : Number(vaultDecimals ?? 24);
  const parsed = useMemo(() => tryParse(input, inputDecimals), [input, inputDecimals]);

  const needsApproval =
    mode === "stake" && parsed !== null && (allowance ?? 0n) < parsed;

  const { writeContractAsync, data: txHash, isPending: isWriting, reset } = useWriteContract();
  const { isLoading: isMining, isSuccess: mined } = useWaitForTransactionReceipt({ hash: txHash });

  if (mined) {
    refetchAssetBal();
    refetchShareBal();
    refetchAllowance();
    setTimeout(reset, 100);
  }

  async function onSubmit() {
    if (!address || !parsed) return;
    if (mode === "stake") {
      if (needsApproval) {
        await writeContractAsync({
          address: ASSET_ADDRESS,
          abi: erc20Abi,
          functionName: "approve",
          args: [VAULT_ADDRESS, parsed],
        });
        return;
      }
      await writeContractAsync({
        address: VAULT_ADDRESS,
        abi: vaultAbi,
        functionName: "deposit",
        args: [parsed, address],
      });
      setInput("");
    } else {
      await writeContractAsync({
        address: VAULT_ADDRESS,
        abi: vaultAbi,
        functionName: "redeem",
        args: [parsed, address, address],
      });
      setInput("");
    }
  }

  const buttonLabel = (() => {
    if (!address) return "Connect wallet";
    if (parsed === null || parsed === 0n) return "Enter amount";
    if (isWriting || isMining) return "Pending…";
    if (mode === "stake" && needsApproval) return `Approve ${assetSymbol ?? "token"}`;
    return mode === "stake" ? "Stake" : "Unstake";
  })();

  const balance = mode === "stake" ? assetBalance : shareBalance;
  const balanceLabel = mode === "stake" ? assetSymbol ?? "Asset" : "Shares";

  return (
    <div className="panel">
      <div className="flex gap-2 mb-4">
        <button
          className={`flex-1 btn ${mode === "stake" ? "btn-primary" : "btn-ghost"}`}
          onClick={() => setMode("stake")}
        >
          Stake
        </button>
        <button
          className={`flex-1 btn ${mode === "unstake" ? "btn-primary" : "btn-ghost"}`}
          onClick={() => setMode("unstake")}
        >
          Unstake
        </button>
      </div>

      <div className="space-y-3">
        <div className="flex justify-between text-xs text-white/50">
          <span>Amount</span>
          <span>
            Balance:{" "}
            <button
              className="font-mono hover:text-white"
              onClick={() => balance !== undefined && setInput(fmt(balance, inputDecimals, 6))}
            >
              {fmt(balance, inputDecimals, 4)} {balanceLabel}
            </button>
          </span>
        </div>
        <input
          className="input"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="0.0"
          inputMode="decimal"
        />

        <button
          className="w-full btn-primary"
          disabled={!address || parsed === null || parsed === 0n || isWriting || isMining}
          onClick={onSubmit}
        >
          {buttonLabel}
        </button>

        {txHash && (
          <div className="text-xs text-white/40 break-all">
            tx: {txHash} {mined && "✓"}
          </div>
        )}
      </div>
    </div>
  );
}
