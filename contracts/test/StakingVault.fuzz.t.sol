// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {StakingVault} from "../src/StakingVault.sol";
import {MockERC20} from "./mocks/MockERC20.sol";

/// @notice Fuzz tests for ERC-4626 math properties of the StakingVault.
contract StakingVaultFuzzTest is Test {
    StakingVault internal vault;
    MockERC20 internal asset;

    address internal admin = makeAddr("admin");
    address internal operator = makeAddr("operator");
    address internal user = makeAddr("user");

    function setUp() public {
        asset = new MockERC20("Mock", "MOCK", 18);
        vault = new StakingVault(IERC20(address(asset)), "stMOCK", "stMOCK", admin, operator);
    }

    /// @notice deposit -> redeem all should return roughly the same amount, never more.
    function testFuzz_DepositRedeem_NoFreeAssets(uint256 amount) public {
        amount = bound(amount, 1, type(uint128).max);

        asset.mint(user, amount);
        vm.prank(user);
        asset.approve(address(vault), amount);

        vm.prank(user);
        uint256 shares = vault.deposit(amount, user);

        vm.prank(user);
        uint256 assetsBack = vault.redeem(shares, user, user);

        // User must never extract more than they put in.
        assertLe(assetsBack, amount, "User extracted more than deposited");
    }

    /// @notice convertToShares and convertToAssets should round-trip without inflating value.
    function testFuzz_ConvertRoundTrip_NoInflation(uint256 assets) public {
        assets = bound(assets, 1, type(uint128).max);

        // Seed vault with some state
        asset.mint(user, 1000 ether);
        vm.prank(user);
        asset.approve(address(vault), type(uint256).max);
        vm.prank(user);
        vault.deposit(1000 ether, user);

        uint256 shares = vault.convertToShares(assets);
        uint256 assetsBack = vault.convertToAssets(shares);

        // Round-trip cannot inflate the asset value.
        assertLe(assetsBack, assets, "Round-trip inflated asset value");
    }

    /// @notice Reward distribution monotonically increases share/asset rate.
    function testFuzz_DistributeRewards_MonotonicallyIncreasesShareValue(uint256 deposit, uint256 reward) public {
        deposit = bound(deposit, 1 ether, type(uint96).max);
        reward = bound(reward, 1, type(uint96).max);

        asset.mint(user, deposit);
        vm.prank(user);
        asset.approve(address(vault), type(uint256).max);
        vm.prank(user);
        vault.deposit(deposit, user);

        uint256 priceBefore = vault.convertToAssets(1e24);

        asset.mint(operator, reward);
        vm.prank(operator);
        asset.approve(address(vault), reward);
        vm.prank(operator);
        vault.distributeRewards(reward);

        uint256 priceAfter = vault.convertToAssets(1e24);
        assertGe(priceAfter, priceBefore, "Reward distribution decreased share value");
    }

    /// @notice previewDeposit must equal actual deposit shares (4626 invariant).
    function testFuzz_PreviewDepositMatchesDeposit(uint256 amount) public {
        amount = bound(amount, 1, type(uint96).max);
        asset.mint(user, amount);
        vm.prank(user);
        asset.approve(address(vault), amount);

        uint256 expected = vault.previewDeposit(amount);
        vm.prank(user);
        uint256 actual = vault.deposit(amount, user);

        assertEq(actual, expected, "previewDeposit lied");
    }

    /// @notice previewRedeem must equal actual redeem assets (4626 invariant).
    function testFuzz_PreviewRedeemMatchesRedeem(uint256 amount) public {
        amount = bound(amount, 1 ether, type(uint96).max);
        asset.mint(user, amount);
        vm.prank(user);
        asset.approve(address(vault), amount);
        vm.prank(user);
        uint256 shares = vault.deposit(amount, user);

        uint256 expected = vault.previewRedeem(shares);
        vm.prank(user);
        uint256 actual = vault.redeem(shares, user, user);

        assertEq(actual, expected, "previewRedeem lied");
    }
}
