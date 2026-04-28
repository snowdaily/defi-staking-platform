import { getDefaultConfig } from "@rainbow-me/rainbowkit";
import { http } from "viem";
import { mainnet, sepolia, baseSepolia, foundry } from "wagmi/chains";

const projectId = process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID || "demo-project-id";
const rpcUrl = process.env.NEXT_PUBLIC_RPC_URL || "http://localhost:8545";

export const wagmiConfig = getDefaultConfig({
  appName: "Liquid Staking",
  projectId,
  chains: [foundry, sepolia, baseSepolia, mainnet],
  transports: {
    [foundry.id]: http(rpcUrl),
    [sepolia.id]: http(),
    [baseSepolia.id]: http(),
    [mainnet.id]: http(),
  },
  ssr: true,
});
