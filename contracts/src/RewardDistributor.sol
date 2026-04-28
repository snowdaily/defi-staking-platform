// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {AccessControl} from "@openzeppelin/contracts/access/AccessControl.sol";
import {ReentrancyGuard} from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import {StakingVault} from "./StakingVault.sol";

/// @title RewardDistributor
/// @notice Holds reward funds and pushes them to the StakingVault on operator
///         command. Lets the off-chain reward bot fund and distribute in
///         separate transactions, and lets the admin recover stray tokens.
contract RewardDistributor is AccessControl, ReentrancyGuard {
    using SafeERC20 for IERC20;

    bytes32 public constant OPERATOR_ROLE = keccak256("OPERATOR_ROLE");

    StakingVault public immutable vault;
    IERC20 public immutable asset;

    error ZeroAddress();
    error ZeroAmount();
    error InsufficientFunds();
    error CannotRescueAsset();

    event Funded(address indexed from, uint256 amount);
    event Distributed(address indexed operator, uint256 amount);
    event Rescued(address indexed token, address indexed to, uint256 amount);

    constructor(StakingVault vault_, address admin_, address operator_) {
        if (address(vault_) == address(0) || admin_ == address(0) || operator_ == address(0)) {
            revert ZeroAddress();
        }
        vault = vault_;
        asset = IERC20(vault_.asset());
        _grantRole(DEFAULT_ADMIN_ROLE, admin_);
        _grantRole(OPERATOR_ROLE, operator_);

        // Pre-approve the vault to pull from this contract during distribute().
        asset.forceApprove(address(vault_), type(uint256).max);
    }

    /// @notice Pull `amount` of the underlying asset from the caller into this contract.
    function fund(uint256 amount) external nonReentrant {
        if (amount == 0) revert ZeroAmount();
        asset.safeTransferFrom(msg.sender, address(this), amount);
        emit Funded(msg.sender, amount);
    }

    /// @notice Distribute `amount` of held funds to the vault, raising share value.
    function distribute(uint256 amount) external onlyRole(OPERATOR_ROLE) nonReentrant {
        if (amount == 0) revert ZeroAmount();
        if (asset.balanceOf(address(this)) < amount) revert InsufficientFunds();
        vault.distributeRewards(amount);
        emit Distributed(msg.sender, amount);
    }

    /// @notice Distribute the full held balance to the vault.
    function distributeAll() external onlyRole(OPERATOR_ROLE) nonReentrant returns (uint256 amount) {
        amount = asset.balanceOf(address(this));
        if (amount == 0) revert ZeroAmount();
        vault.distributeRewards(amount);
        emit Distributed(msg.sender, amount);
    }

    /// @notice Currently held, undistributed reward balance.
    function pendingRewards() external view returns (uint256) {
        return asset.balanceOf(address(this));
    }

    /// @notice Recover stray tokens accidentally sent to this contract.
    /// @dev    The underlying asset cannot be rescued — use distribute / distributeAll.
    function rescue(IERC20 token, address to, uint256 amount) external onlyRole(DEFAULT_ADMIN_ROLE) {
        if (address(token) == address(asset)) revert CannotRescueAsset();
        token.safeTransfer(to, amount);
        emit Rescued(address(token), to, amount);
    }
}
