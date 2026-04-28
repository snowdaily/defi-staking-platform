// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {StakingVault} from "../src/StakingVault.sol";
import {MockERC20} from "./mocks/MockERC20.sol";

/// @notice Handler the invariant fuzzer drives. Bounds inputs so the fuzzer
///         hits realistic states instead of reverting on every call.
contract StakingVaultHandler is Test {
    StakingVault public immutable vault;
    MockERC20 public immutable asset;
    address public immutable operator;

    address[] public actors;
    uint256 public totalDeposited;
    uint256 public totalWithdrawn;
    uint256 public totalDistributed;

    constructor(StakingVault vault_, MockERC20 asset_, address operator_, address[] memory actors_) {
        vault = vault_;
        asset = asset_;
        operator = operator_;
        actors = actors_;
    }

    function _actor(uint256 idx) internal view returns (address) {
        return actors[idx % actors.length];
    }

    function deposit(uint256 actorSeed, uint256 amount) external {
        amount = bound(amount, 1, 1_000_000 ether);
        address actor = _actor(actorSeed);
        asset.mint(actor, amount);
        vm.startPrank(actor);
        asset.approve(address(vault), amount);
        vault.deposit(amount, actor);
        vm.stopPrank();
        totalDeposited += amount;
    }

    function withdraw(uint256 actorSeed, uint256 sharesPct) external {
        address actor = _actor(actorSeed);
        uint256 shares = vault.balanceOf(actor);
        if (shares == 0) return;
        sharesPct = bound(sharesPct, 1, 100);
        uint256 sharesToBurn = (shares * sharesPct) / 100;
        if (sharesToBurn == 0) return;
        vm.startPrank(actor);
        uint256 out = vault.redeem(sharesToBurn, actor, actor);
        vm.stopPrank();
        totalWithdrawn += out;
    }

    function distribute(uint256 amount) external {
        amount = bound(amount, 1, 100_000 ether);
        asset.mint(operator, amount);
        vm.startPrank(operator);
        asset.approve(address(vault), amount);
        vault.distributeRewards(amount);
        vm.stopPrank();
        totalDistributed += amount;
    }
}

/// @notice Invariant tests: properties that must hold across any sequence of operations.
contract StakingVaultInvariantTest is Test {
    StakingVault internal vault;
    MockERC20 internal asset;
    StakingVaultHandler internal handler;

    address internal admin = makeAddr("admin");
    address internal operator = makeAddr("operator");

    function setUp() public {
        asset = new MockERC20("Mock", "MOCK", 18);
        vault = new StakingVault(IERC20(address(asset)), "stMOCK", "stMOCK", admin, operator);

        address[] memory actors = new address[](4);
        actors[0] = makeAddr("a1");
        actors[1] = makeAddr("a2");
        actors[2] = makeAddr("a3");
        actors[3] = makeAddr("a4");

        handler = new StakingVaultHandler(vault, asset, operator, actors);

        targetContract(address(handler));

        // Limit the surface area the fuzzer touches.
        bytes4[] memory selectors = new bytes4[](3);
        selectors[0] = StakingVaultHandler.deposit.selector;
        selectors[1] = StakingVaultHandler.withdraw.selector;
        selectors[2] = StakingVaultHandler.distribute.selector;
        targetSelector(FuzzSelector({addr: address(handler), selectors: selectors}));
    }

    /// @notice Underlying assets held by the vault must always equal totalAssets().
    function invariant_VaultBalanceMatchesTotalAssets() public view {
        assertEq(asset.balanceOf(address(vault)), vault.totalAssets());
    }

    /// @notice The vault must be able to satisfy every share holder simultaneously.
    function invariant_TotalAssetsCoversAllShares() public view {
        uint256 sumOfClaims = vault.convertToAssets(vault.totalSupply());
        // convertToAssets rounds down → totalAssets must be >= sum of claims
        assertGe(vault.totalAssets(), sumOfClaims, "Vault undercollateralised");
    }

    /// @notice Cumulative deposits + distributions = totalAssets + cumulative withdrawals.
    function invariant_AssetsConserved() public view {
        assertEq(
            handler.totalDeposited() + handler.totalDistributed(),
            vault.totalAssets() + handler.totalWithdrawn(),
            "Asset conservation broken"
        );
    }
}
