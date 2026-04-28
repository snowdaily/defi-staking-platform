// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {IAccessControl} from "@openzeppelin/contracts/access/IAccessControl.sol";
import {RewardDistributor} from "../src/RewardDistributor.sol";
import {StakingVault} from "../src/StakingVault.sol";
import {MockERC20} from "./mocks/MockERC20.sol";

contract RewardDistributorTest is Test {
    StakingVault internal vault;
    MockERC20 internal asset;
    RewardDistributor internal distributor;

    address internal admin = makeAddr("admin");
    address internal operator = makeAddr("operator");
    address internal funder = makeAddr("funder");
    address internal alice = makeAddr("alice");

    uint256 internal constant FUND_AMOUNT = 100_000 ether;

    function setUp() public {
        asset = new MockERC20("Mock", "MOCK", 18);
        vault = new StakingVault(IERC20(address(asset)), "stMOCK", "stMOCK", admin, operator);

        distributor = new RewardDistributor(vault, admin, operator);

        // Grant the distributor OPERATOR_ROLE on the vault so it can call distributeRewards.
        bytes32 vaultOpRole = vault.OPERATOR_ROLE();
        vm.prank(admin);
        vault.grantRole(vaultOpRole, address(distributor));

        // Seed funder with assets and have them fund the distributor.
        asset.mint(funder, FUND_AMOUNT);
        vm.prank(funder);
        asset.approve(address(distributor), type(uint256).max);

        // Alice deposits into the vault so distributions have someone to benefit.
        asset.mint(alice, 10_000 ether);
        vm.prank(alice);
        asset.approve(address(vault), type(uint256).max);
        vm.prank(alice);
        vault.deposit(1_000 ether, alice);
    }

    /*//////////////////////////////////////////////////////////////
                              CONSTRUCTOR
    //////////////////////////////////////////////////////////////*/

    function test_Constructor_StoresVaultAndAsset() public view {
        assertEq(address(distributor.vault()), address(vault));
        assertEq(address(distributor.asset()), address(asset));
    }

    function test_Constructor_GrantsRoles() public view {
        assertTrue(distributor.hasRole(distributor.DEFAULT_ADMIN_ROLE(), admin));
        assertTrue(distributor.hasRole(distributor.OPERATOR_ROLE(), operator));
    }

    function test_Constructor_RevertsOnZeroVault() public {
        vm.expectRevert(RewardDistributor.ZeroAddress.selector);
        new RewardDistributor(StakingVault(address(0)), admin, operator);
    }

    function test_Constructor_RevertsOnZeroAdmin() public {
        vm.expectRevert(RewardDistributor.ZeroAddress.selector);
        new RewardDistributor(vault, address(0), operator);
    }

    function test_Constructor_RevertsOnZeroOperator() public {
        vm.expectRevert(RewardDistributor.ZeroAddress.selector);
        new RewardDistributor(vault, admin, address(0));
    }

    /*//////////////////////////////////////////////////////////////
                                 FUND
    //////////////////////////////////////////////////////////////*/

    function test_Fund_PullsAssetIntoDistributor() public {
        vm.prank(funder);
        distributor.fund(500 ether);
        assertEq(asset.balanceOf(address(distributor)), 500 ether);
        assertEq(distributor.pendingRewards(), 500 ether);
    }

    function test_Fund_RevertsOnZero() public {
        vm.prank(funder);
        vm.expectRevert(RewardDistributor.ZeroAmount.selector);
        distributor.fund(0);
    }

    /*//////////////////////////////////////////////////////////////
                              DISTRIBUTE
    //////////////////////////////////////////////////////////////*/

    function test_Distribute_PushesAssetToVault_RaisesShareValue() public {
        vm.prank(funder);
        distributor.fund(500 ether);

        uint256 priceBefore = vault.convertToAssets(1 ether);
        uint256 vaultAssetsBefore = vault.totalAssets();

        vm.prank(operator);
        distributor.distribute(500 ether);

        assertEq(vault.totalAssets(), vaultAssetsBefore + 500 ether);
        assertGt(vault.convertToAssets(1 ether), priceBefore);
        assertEq(asset.balanceOf(address(distributor)), 0);
    }

    function test_Distribute_PartialAmount() public {
        vm.prank(funder);
        distributor.fund(500 ether);

        vm.prank(operator);
        distributor.distribute(200 ether);

        assertEq(asset.balanceOf(address(distributor)), 300 ether);
        assertEq(distributor.pendingRewards(), 300 ether);
    }

    function test_Distribute_RevertsForNonOperator() public {
        bytes32 opRole = distributor.OPERATOR_ROLE();
        vm.prank(funder);
        distributor.fund(500 ether);
        vm.prank(alice);
        vm.expectRevert(
            abi.encodeWithSelector(IAccessControl.AccessControlUnauthorizedAccount.selector, alice, opRole)
        );
        distributor.distribute(100 ether);
    }

    function test_Distribute_RevertsOnZero() public {
        vm.prank(operator);
        vm.expectRevert(RewardDistributor.ZeroAmount.selector);
        distributor.distribute(0);
    }

    function test_Distribute_RevertsWhenAmountExceedsBalance() public {
        vm.prank(funder);
        distributor.fund(100 ether);
        vm.prank(operator);
        vm.expectRevert(RewardDistributor.InsufficientFunds.selector);
        distributor.distribute(200 ether);
    }

    /*//////////////////////////////////////////////////////////////
                          DISTRIBUTE-ALL HELPER
    //////////////////////////////////////////////////////////////*/

    function test_DistributeAll_DrainsBalance() public {
        vm.prank(funder);
        distributor.fund(750 ether);
        vm.prank(operator);
        uint256 distributed = distributor.distributeAll();
        assertEq(distributed, 750 ether);
        assertEq(asset.balanceOf(address(distributor)), 0);
    }

    function test_DistributeAll_RevertsWhenEmpty() public {
        vm.prank(operator);
        vm.expectRevert(RewardDistributor.ZeroAmount.selector);
        distributor.distributeAll();
    }

    /*//////////////////////////////////////////////////////////////
                                RESCUE
    //////////////////////////////////////////////////////////////*/

    function test_Rescue_AdminCanRecoverNonAssetTokens() public {
        MockERC20 stray = new MockERC20("Stray", "S", 18);
        stray.mint(address(distributor), 999 ether);

        vm.prank(admin);
        distributor.rescue(IERC20(address(stray)), admin, 999 ether);

        assertEq(stray.balanceOf(admin), 999 ether);
    }

    function test_Rescue_CannotTakeUnderlyingAsset() public {
        vm.prank(funder);
        distributor.fund(500 ether);

        vm.prank(admin);
        vm.expectRevert(RewardDistributor.CannotRescueAsset.selector);
        distributor.rescue(IERC20(address(asset)), admin, 100 ether);
    }

    function test_Rescue_RevertsForNonAdmin() public {
        bytes32 adminRole = distributor.DEFAULT_ADMIN_ROLE();
        MockERC20 stray = new MockERC20("Stray", "S", 18);
        stray.mint(address(distributor), 100 ether);

        vm.prank(alice);
        vm.expectRevert(
            abi.encodeWithSelector(IAccessControl.AccessControlUnauthorizedAccount.selector, alice, adminRole)
        );
        distributor.rescue(IERC20(address(stray)), alice, 100 ether);
    }
}
