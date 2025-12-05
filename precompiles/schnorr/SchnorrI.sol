// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev Schnorr precompile address.
address constant SCHNORR_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000703;

/// @dev Schnorr precompile interface bound to the well-known address.
/// @notice Verifies BIP-340 Schnorr signatures (x-only pubkey, 32-byte hash, 64-byte signature).
interface SchnorrI {
    /**
     * @notice Verify a BIP-340 Schnorr signature.
     * @param xOnlyPubKey   32-byte x-only public key (taproot-style).
     * @param signature     64-byte Schnorr signature.
     * @param messageHash   32-byte message hash (already hashed).
     * @return success      true iff the signature is valid for (xOnlyPubKey, messageHash).
     */
    function verifySchnorrSignature(
        bytes32 xOnlyPubKey,
        bytes calldata signature,
        bytes32 messageHash
    ) external pure returns (bool success);
}

SchnorrI constant SCHNORR_CONTRACT = SchnorrI(SCHNORR_PRECOMPILE_ADDRESS);
