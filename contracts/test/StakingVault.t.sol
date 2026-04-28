// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {IAccessControl} from "@openzeppelin/contracts/access/IAccessControl.sol";
import {Pausable} from "@openzeppelin/contracts/utils/Pausable.sol";
import {StakingVault} from "../src/StakingVault.sol";
import {MockERC20} from "./mocks/MockERC20.sol";

/// @notice TDD test suite for the StakingVault.
/// Tests are written before the implementation and define expected behaviour.
contract StakingVaultTest is Test {
    StakingVault internal vault;
    MockERC20 internal asset;

    address internal admin = makeAddr("admin");
    address internal operator = makeAddr("operator");
    address internal alice = makeAddr("alice");
    address internal bob = makeAddr("bob");
    address internal attacker = makeAddr("attacker");

    uint256 internal constant INITIAL_MINT = 1_000_000 ether;

    function setUp() public {
        asset = new MockERC20("Mock Token", "MOCK", 18);

        vault = new StakingVault(IERC20(address(asset)), "Staked Mock", "stMOCK", admin, operator);

        asset.mint(alice, INITIAL_MINT);
        asset.mint(bob, INITIAL_MINT);
        asset.mint(attacker, INITIAL_MINT);
        asset.mint(operator, INITIAL_MINT);

        vm.prank(alice);
        asset.approve(address(vault), type(uint256).max);
        vm.prank(bob);
        asset.approve(address(vault), type(uint256).max);
        vm.prank(attacker);
        asset.approve(address(vault), type(uint256).max);
        vm.prank(operator);
        asset.approve(address(vault), type(uint256).max);
    }

    /*//////////////////////////////////////////////////////////////
                              CONSTRUCTOR
    //////////////////////////////////////////////////////////////*/

    function test_Constructor_SetsAssetAndMetadata() public view {
        assertEq(address(vault.asset()), address(asset));
        assertEq(vault.name(), "Staked Mock");
        assertEq(vault.symbol(), "stMOCK");
    }

    function test_Constructor_GrantsAdminAndOperatorRoles() public view {
        assertTrue(vault.hasRole(vault.DEFAULT_ADMIN_ROLE(), admin));
        assertTrue(vault.hasRole(vault.OPERATOR_ROLE(), operator));
    }

    function test_Constructor_RevertsOnZeroAdmin() public {
        vm.expectRevert(StakingVault.ZeroAddress.selector);
        new StakingVault(IERC20(address(asset)), "x", "x", address(0), operator);
    }

    function test_Constructor_RevertsOnZeroOperator() public {
        vm.expectRevert(StakingVault.ZeroAddress.selector);
        new StakingVault(IERC20(address(asset)), "x", "x", admin, address(0));
    }

    /*//////////////////////////////////////////////////////////////
                                DEPOSIT
    //////////////////////////////////////////////////////////////*/

    function test_Deposit_FirstDepositorMintsSharesOneToOne() public {
        uint256 amount = 100 ether;
        vm.prank(alice);
        uint256 shares = vault.deposit(amount, alice);

        // OZ ERC-4626 with virtual offset still keeps roughly 1:1 for the depositor.
        assertEq(vault.balanceOf(alice), shares);
        assertEq(vault.totalAssets(), amount);
        assertEq(asset.balanceOf(address(vault)), amount);
    }

    function test_Deposit_RevertsOnZeroAmount() public {
        vm.prank(alice);
        vm.expectRevert(StakingVault.ZeroAmount.selector);
        vault.deposit(0, alice);
    }

    function test_Deposit_RevertsWhenPaused() public {
        vm.prank(admin);
        vault.pause();

        vm.prank(alice);
        vm.expectRevert(Pausable.EnforcedPause.selector);
        vault.deposit(100 ether, alice);
    }

    /*//////////////////////////////////////////////////////////////
                               WITHDRAW
    //////////////////////////////////////////////////////////////*/

    function test_Withdraw_ReturnsAssetsAndBurnsShares() public {
        uint256 amount = 100 ether;
        vm.prank(alice);
        vault.deposit(amount, alice);

        uint256 sharesBefore = vault.balanceOf(alice);
        uint256 assetsBefore = asset.balanceOf(alice);

        vm.prank(alice);
        vault.withdraw(amount, alice, alice);

        assertEq(asset.balanceOf(alice), assetsBefore + amount);
        assertLt(vault.balanceOf(alice), sharesBefore);
    }

    function test_Withdraw_RevertsOnZeroAmount() public {
        vm.prank(alice);
        vault.deposit(100 ether, alice);

        vm.prank(alice);
        vm.expectRevert(StakingVault.ZeroAmount.selector);
        vault.withdraw(0, alice, alice);
    }

    function test_Redeem_BurnsSharesReturnsAssets() public {
        uint256 amount = 100 ether;
        vm.prank(alice);
        uint256 shares = vault.deposit(amount, alice);

        vm.prank(alice);
        uint256 assetsOut = vault.redeem(shares, alice, alice);

        assertEq(vault.balanceOf(alice), 0);
        assertApproxEqAbs(assetsOut, amount, 1); // rounding tolerance from virtual shares
    }

    /*//////////////////////////////////////////////////////////////
                          REWARD DISTRIBUTION
    //////////////////////////////////////////////////////////////*/

    function test_DistributeRewards_IncreasesShareValue() public {
        uint256 deposit = 1000 ether;
        vm.prank(alice);
        vault.deposit(deposit, alice);

        uint256 priceBefore = vault.convertToAssets(1 ether);

        uint256 reward = 100 ether;
        vm.prank(operator);
        vault.distributeRewards(reward);

        uint256 priceAfter = vault.convertToAssets(1 ether);
        assertGt(priceAfter, priceBefore, "Share price must rise after reward");
        assertEq(vault.totalAssets(), deposit + reward);
    }

    function test_DistributeRewards_RevertsForNonOperator() public {
        bytes32 operatorRole = vault.OPERATOR_ROLE();
        vm.prank(alice);
        asset.approve(address(vault), 100 ether);
        vm.prank(alice);
        vm.expectRevert(
            abi.encodeWithSelector(IAccessControl.AccessControlUnauthorizedAccount.selector, alice, operatorRole)
        );
        vault.distributeRewards(100 ether);
    }

    function test_DistributeRewards_RevertsOnZeroAmount() public {
        vm.prank(operator);
        vm.expectRevert(StakingVault.ZeroAmount.selector);
        vault.distributeRewards(0);
    }

    function test_LateDepositor_GetsFewerSharesAfterRewards() public {
        // Alice deposits first, gets baseline shares
        vm.prank(alice);
        vault.deposit(1000 ether, alice);
        uint256 aliceShares = vault.balanceOf(alice);

        // Reward distributed
        vm.prank(operator);
        vault.distributeRewards(500 ether);

        // Bob deposits the same amount Alice did
        vm.prank(bob);
        vault.deposit(1000 ether, bob);
        uint256 bobShares = vault.balanceOf(bob);

        // Bob gets fewer shares for the same asset amount because share price went up
        assertLt(bobShares, aliceShares, "Late depositor must receive fewer shares");
    }

    /*//////////////////////////////////////////////////////////////
                       INFLATION ATTACK DEFENSE
    //////////////////////////////////////////////////////////////*/

    function test_FirstDepositorInflationAttack_VictimGetsFairShares() public {
        // Attacker deposits 1 wei to get 1 share, then donates a huge amount
        // hoping the next depositor will get 0 shares due to rounding.
        vm.prank(attacker);
        vault.deposit(1, attacker);

        // Donation: send tokens directly to the vault without depositing
        vm.prank(attacker);
        asset.transfer(address(vault), 10_000 ether);

        // Victim deposits a normal amount
        vm.prank(alice);
        uint256 victimShares = vault.deposit(1000 ether, alice);

        // With OZ virtual-shares defense, victim should still receive a non-trivial amount
        // (the donation gets diluted across the virtual offset).
        assertGt(victimShares, 0, "Victim must receive shares despite donation");

        // Victim's redemption value must be within 0.1% of their deposit.
        // A regression that weakened the defense (e.g., offset=2) would fail here.
        uint256 redeemable = vault.convertToAssets(victimShares);
        assertApproxEqRel(redeemable, 1000 ether, 1e15, "Victim redemption value below 99.9% of deposit");
    }

    /*//////////////////////////////////////////////////////////////
                                PAUSE
    //////////////////////////////////////////////////////////////*/

    function test_Pause_OnlyAdmin() public {
        bytes32 adminRole = vault.DEFAULT_ADMIN_ROLE();
        vm.prank(alice);
        vm.expectRevert(
            abi.encodeWithSelector(IAccessControl.AccessControlUnauthorizedAccount.selector, alice, adminRole)
        );
        vault.pause();
    }

    function test_Unpause_RestoresDeposits() public {
        vm.prank(admin);
        vault.pause();

        vm.prank(admin);
        vault.unpause();

        vm.prank(alice);
        vault.deposit(100 ether, alice); // must succeed
        assertGt(vault.balanceOf(alice), 0);
    }

    /*//////////////////////////////////////////////////////////////
                         EMERGENCY WITHDRAW
    //////////////////////////////////////////////////////////////*/

    function test_EmergencyWithdraw_WorksWhenPaused() public {
        vm.prank(alice);
        vault.deposit(100 ether, alice);
        uint256 sharesBefore = vault.balanceOf(alice);

        vm.prank(admin);
        vault.pause();

        vm.prank(alice);
        uint256 assetsOut = vault.emergencyWithdraw(sharesBefore);

        assertEq(vault.balanceOf(alice), 0);
        assertGt(assetsOut, 0);
        assertGt(asset.balanceOf(alice), INITIAL_MINT - 100 ether);
    }

    function test_EmergencyWithdraw_RevertsOnZeroShares() public {
        vm.prank(alice);
        vm.expectRevert(StakingVault.ZeroAmount.selector);
        vault.emergencyWithdraw(0);
    }

    /*//////////////////////////////////////////////////////////////
                             ACCOUNTING
    //////////////////////////////////////////////////////////////*/

    function test_Mint_GivesExactShares() public {
        uint256 shares = 50 ether;
        vm.prank(alice);
        uint256 assetsIn = vault.mint(shares, alice);

        assertEq(vault.balanceOf(alice), shares);
        assertGt(assetsIn, 0);
    }

    function test_Mint_RevertsOnZero() public {
        vm.prank(alice);
        vm.expectRevert(StakingVault.ZeroAmount.selector);
        vault.mint(0, alice);
    }

    function test_Mint_RevertsWhenPaused() public {
        vm.prank(admin);
        vault.pause();

        vm.prank(alice);
        vm.expectRevert(Pausable.EnforcedPause.selector);
        vault.mint(1 ether, alice);
    }

    function test_Decimals_ReportsAssetDecimalsPlusOffset() public view {
        // Asset is 18 decimals, _decimalsOffset is 6 → vault decimals = 24
        assertEq(vault.decimals(), 18 + 6);
    }

    function test_TotalAssets_TracksDepositsAndRewards() public {
        vm.prank(alice);
        vault.deposit(100 ether, alice);
        vm.prank(bob);
        vault.deposit(200 ether, bob);
        assertEq(vault.totalAssets(), 300 ether);

        vm.prank(operator);
        vault.distributeRewards(50 ether);
        assertEq(vault.totalAssets(), 350 ether);
    }
}
