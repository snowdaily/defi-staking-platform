// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import {ERC4626} from "@openzeppelin/contracts/token/ERC20/extensions/ERC4626.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {IERC20Metadata} from "@openzeppelin/contracts/token/ERC20/extensions/IERC20Metadata.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {AccessControl} from "@openzeppelin/contracts/access/AccessControl.sol";
import {Pausable} from "@openzeppelin/contracts/utils/Pausable.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/// @title StakingVault
/// @notice ERC-4626 vault that issues a yield-bearing liquid staking token.
///         Reward-bearing model: distributing rewards increases the share/asset
///         exchange rate; share balances do not change (no rebasing).
/// @dev    Inflation attack ("first depositor donation") is mitigated via OZ's
///         virtual-shares offset (`_decimalsOffset`).
contract StakingVault is ERC4626, AccessControl, Pausable, ReentrancyGuard {
    using SafeERC20 for IERC20;

    /*//////////////////////////////////////////////////////////////
                                ROLES
    //////////////////////////////////////////////////////////////*/

    bytes32 public constant OPERATOR_ROLE = keccak256("OPERATOR_ROLE");

    /*//////////////////////////////////////////////////////////////
                                ERRORS
    //////////////////////////////////////////////////////////////*/

    error ZeroAddress();
    error ZeroAmount();

    /*//////////////////////////////////////////////////////////////
                                EVENTS
    //////////////////////////////////////////////////////////////*/

    event RewardsDistributed(address indexed operator, uint256 amount, uint256 totalAssetsAfter);
    event EmergencyWithdraw(address indexed user, uint256 shares, uint256 assets);

    /*//////////////////////////////////////////////////////////////
                              CONSTRUCTOR
    //////////////////////////////////////////////////////////////*/

    constructor(IERC20 asset_, string memory name_, string memory symbol_, address admin_, address operator_)
        ERC20(name_, symbol_)
        ERC4626(asset_)
    {
        if (admin_ == address(0) || operator_ == address(0)) revert ZeroAddress();
        _grantRole(DEFAULT_ADMIN_ROLE, admin_);
        _grantRole(OPERATOR_ROLE, operator_);
    }

    /*//////////////////////////////////////////////////////////////
                          INFLATION DEFENSE
    //////////////////////////////////////////////////////////////*/

    /// @dev Virtual share offset of 6 makes the donation attack
    ///      uneconomical (attacker would need to donate ~1e6 of inflation).
    function _decimalsOffset() internal pure override returns (uint8) {
        return 6;
    }

    /*//////////////////////////////////////////////////////////////
                          ERC-4626 OVERRIDES
    //////////////////////////////////////////////////////////////*/

    function deposit(uint256 assets, address receiver) public override whenNotPaused nonReentrant returns (uint256) {
        if (assets == 0) revert ZeroAmount();
        return super.deposit(assets, receiver);
    }

    function mint(uint256 shares, address receiver) public override whenNotPaused nonReentrant returns (uint256) {
        if (shares == 0) revert ZeroAmount();
        return super.mint(shares, receiver);
    }

    function withdraw(uint256 assets, address receiver, address owner)
        public
        override
        whenNotPaused
        nonReentrant
        returns (uint256)
    {
        if (assets == 0) revert ZeroAmount();
        return super.withdraw(assets, receiver, owner);
    }

    function redeem(uint256 shares, address receiver, address owner)
        public
        override
        whenNotPaused
        nonReentrant
        returns (uint256)
    {
        if (shares == 0) revert ZeroAmount();
        return super.redeem(shares, receiver, owner);
    }

    /*//////////////////////////////////////////////////////////////
                         REWARD DISTRIBUTION
    //////////////////////////////////////////////////////////////*/

    /// @notice Pulls `amount` of the underlying asset into the vault, raising
    ///         the share/asset exchange rate for all current holders.
    /// @dev    Caller must hold OPERATOR_ROLE and have approved this contract
    ///         for at least `amount` of the underlying asset.
    function distributeRewards(uint256 amount) external onlyRole(OPERATOR_ROLE) nonReentrant {
        if (amount == 0) revert ZeroAmount();
        IERC20(asset()).safeTransferFrom(msg.sender, address(this), amount);
        emit RewardsDistributed(msg.sender, amount, totalAssets());
    }

    /*//////////////////////////////////////////////////////////////
                                PAUSE
    //////////////////////////////////////////////////////////////*/

    function pause() external onlyRole(DEFAULT_ADMIN_ROLE) {
        _pause();
    }

    function unpause() external onlyRole(DEFAULT_ADMIN_ROLE) {
        _unpause();
    }

    /*//////////////////////////////////////////////////////////////
                         EMERGENCY WITHDRAW
    //////////////////////////////////////////////////////////////*/

    /// @notice Always-callable redemption path that bypasses pause.
    ///         Protects users from indefinite admin pause.
    /// @param  shares Number of vault shares to burn.
    /// @return assets Amount of underlying asset returned.
    function emergencyWithdraw(uint256 shares) external nonReentrant returns (uint256 assets) {
        if (shares == 0) revert ZeroAmount();
        assets = previewRedeem(shares);
        _burn(msg.sender, shares);
        IERC20(asset()).safeTransfer(msg.sender, assets);
        emit EmergencyWithdraw(msg.sender, shares, assets);
    }

    /*//////////////////////////////////////////////////////////////
                          DECIMALS RESOLUTION
    //////////////////////////////////////////////////////////////*/

    function decimals() public view override(ERC4626) returns (uint8) {
        return ERC4626.decimals();
    }
}
