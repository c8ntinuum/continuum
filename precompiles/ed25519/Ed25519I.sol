// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The Ed25519 contract's address.
address constant Ed25519_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000500;

/// @dev The Ed25519 contract's instance.
Ed25519I constant ED25519_CONTRACT = Ed25519I(Ed25519_PRECOMPILE_ADDRESS);

/// @author c8ntinuum Team
/// @title Ed25519 Precompiled Contract
/// @dev The interface through which solidity contracts can verify Ed25519 signatures
/// @custom:address 0x0000000000000000000000000000000000000500
interface Ed25519I {
    /// @dev Defines a method for verifying Ed25519 signatures
    /// @param ed25519Address The ed25519 public key (address).
    /// @param signature The ed25519 signature bytes.
    /// @param message The ed25519 message signed represented as bytes.
    /// @return success boolean which is true or false if signature verification worked or not.
    function verifyEd25519Signature(
        bytes memory ed25519Address,
        bytes memory signature,
        bytes memory message
    ) external pure returns (bool success);
}
