"use client";

import { useState, useMemo, useEffect, useRef } from "react";
import {
  useAccount,
  useChainId,
  useReadContract,
  useWriteContract,
  useWaitForTransactionReceipt,
} from "wagmi";
import { foundry, sepolia, baseSepolia, mainnet } from "wagmi/chains";
import { erc20Abi, vaultAbi, ASSET_ADDRESS, VAULT_ADDRESS } from "@/lib/contracts";
import { fmt, tryParse } from "@/lib/format";

type Mode = "stake" | "unstake";

const SUPPORTED_CHAINS = [foundry.id, sepolia.id, baseSepolia.id, mainnet.id] as const;

export function StakeCard() {
  const { address } = useAccount();
  const chainId = useChainId();
  const [mode, setMode] = useState<Mode>("stake");
  const [input, setInput] = useState("");
  const pendingDepositRef = useRef<bigint | null>(null);

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

  const needsApproval = mode === "stake" && parsed !== null && (allowance ?? 0n) < parsed;

  const { writeContractAsync, data: txHash, isPending: isWriting, reset } = useWriteContract();
  const { isLoading: isMining, isSuccess: mined } = useWaitForTransactionReceipt({ hash: txHash });

  // Refetch balances and continue the approve→deposit chain after a tx mines.
  // Side effects must live in useEffect, not in render — see React docs on
  // "side effects in render" — re-running infinitely otherwise.
  useEffect(() => {
    if (!mined) return;
    void (async () => {
      await Promise.all([refetchAssetBal(), refetchShareBal(), refetchAllowance()]);

      // If the just-mined tx was an approve, automatically continue with
      // the deposit the user was actually trying to do — this avoids the
      // race where the user clicks "Stake" before the new allowance is
      // visible to the read RPC.
      const pendingDeposit = pendingDepositRef.current;
      if (pendingDeposit !== null && address) {
        pendingDepositRef.current = null;
        try {
          await writeContractAsync({
            address: VAULT_ADDRESS,
            abi: vaultAbi,
            functionName: "deposit",
            args: [pendingDeposit, address],
          });
          setInput("");
        } catch {
          // user-rejected or other write failure — leave UI in idle state
        }
        return;
      }

      reset();
    })();
    // We intentionally depend on `mined` only; downstream queries are stable refs.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mined]);

  async function onSubmit() {
    if (!address || !parsed) return;
    if (mode === "stake") {
      if (needsApproval) {
        // Stash the deposit amount; the useEffect above will auto-deposit
        // once the approve transaction is mined.
        pendingDepositRef.current = parsed;
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

  const wrongNetwork = !!address && !SUPPORTED_CHAINS.includes(chainId as (typeof SUPPORTED_CHAINS)[number]);

  const buttonLabel = (() => {
    if (!address) return "Connect wallet";
    if (wrongNetwork) return "Switch network";
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

        {wrongNetwork && (
          <div className="text-xs text-warn">
            Connected to an unsupported chain — switch to a configured network.
          </div>
        )}

        <button
          className="w-full btn-primary"
          disabled={
            !address || wrongNetwork || parsed === null || parsed === 0n || isWriting || isMining
          }
          onClick={onSubmit}
        >
          {buttonLabel}
        </button>

        {txHash && (
          <div className="text-xs text-white/40 break-all">
            tx: {txHash} {mined && "(confirmed)"}
          </div>
        )}
      </div>
    </div>
  );
}
