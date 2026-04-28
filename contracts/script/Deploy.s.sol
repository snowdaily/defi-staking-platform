// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console2} from "forge-std/Script.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {StakingVault} from "../src/StakingVault.sol";
import {RewardDistributor} from "../src/RewardDistributor.sol";
import {MockERC20} from "../test/mocks/MockERC20.sol";

/// @notice Deploys MockToken + StakingVault + RewardDistributor and wires
///         OPERATOR_ROLE on the vault to the distributor.
///
/// Usage (local Anvil):
///   forge script script/Deploy.s.sol --rpc-url http://localhost:8545 --broadcast
///
/// Usage (Sepolia):
///   forge script script/Deploy.s.sol --rpc-url $SEPOLIA_RPC_URL \
///     --private-key $DEPLOYER_PRIVATE_KEY --broadcast --verify
contract Deploy is Script {
    function run() external returns (MockERC20 asset, StakingVault vault, RewardDistributor distributor) {
        uint256 pk = vm.envOr("DEPLOYER_PRIVATE_KEY", uint256(0));
        address deployer;
        if (pk == 0) {
            // Default Anvil account #0 — only safe for local devnet.
            pk = 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80;
        }
        deployer = vm.addr(pk);

        vm.startBroadcast(pk);

        asset = new MockERC20("Mock USD", "mUSD", 18);
        vault = new StakingVault(IERC20(address(asset)), "Staked Mock USD", "stmUSD", deployer, deployer);
        distributor = new RewardDistributor(vault, deployer, deployer);

        // Wire the distributor as a vault operator so it can call distributeRewards.
        vault.grantRole(vault.OPERATOR_ROLE(), address(distributor));

        // Seed deployer with some initial test funds (only useful on dev networks).
        asset.mint(deployer, 1_000_000 ether);

        vm.stopBroadcast();

        console2.log("Asset       :", address(asset));
        console2.log("Vault       :", address(vault));
        console2.log("Distributor :", address(distributor));
        console2.log("Deployer    :", deployer);
    }
}
