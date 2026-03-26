// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev PQ Stateless Hash-based Digital Signature (SLH-DSA) precompile address.
address constant PQSLHDSA_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000713;

/// @dev PQSLHDSAI precompile interface bound to the well-known address.
/// @notice Verifies PQ SLH-DSA signatures.
/// @dev
///  - `paramId` selects the parameter set (implementation-defined).
///  - `msgHash` is a 32-byte message digest (e.g. SHA-256 / SHA3-256).
///  - `pubkey` and `signature` are scheme-dependent encoded byte strings.
interface PQSLHDSAI {
    /**
     * @notice Verify a PQ SLH-DSA signature.
     * @param paramId   Parameter set identifier.
     * @param msgHash   32-byte message hash.
     * @param pubkey    Encoded public key bytes.
     * @param signature Encoded signature bytes.
     * @return success  true iff the signature is valid for (paramId, msgHash, pubkey).
     */
    function verify(
        uint8 paramId,
        bytes32 msgHash,
        bytes calldata pubkey,
        bytes calldata signature
    ) external view returns (bool success);
}

/// @dev PQSLHDSAI precompile interface instance at the well-known address.
PQSLHDSAI constant PQSLHDSA_CONTRACT = PQSLHDSAI(PQSLHDSA_PRECOMPILE_ADDRESS);
