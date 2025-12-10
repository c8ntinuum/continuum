// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev PQ Module-Lattice Digital Signature precompile address.
address constant PQMLDSA_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000712;

/// @dev PQMLDSAI precompile interface bound to the well-known address.
/// @notice Verifies Module-Lattice Digital Signatures (ML-DSA-44 / 65 / 87).
/// @dev
///  - `scheme` selects the parameter set:
///        44 → ML-DSA-44
///        65 → ML-DSA-65
///        87 → ML-DSA-87
///  - `msgHash` is a 32-byte message digest (e.g. SHA-256 / SHA3-256)
///  - `pubkey` and `signature` are scheme-dependent encoded byte strings.
interface PQMLDSAI {
    /**
     * @notice Verify a PQ ML-DSA signature.
     * @param scheme    Parameter set identifier (44, 65, 87).
     * @param msgHash   32-byte message hash.
     * @param pubkey    Encoded public key bytes.
     * @param signature Encoded signature bytes.
     * @return success  true iff the signature is valid for (scheme, msgHash, pubkey).
     */
    function verify(
        uint8 scheme,
        bytes32 msgHash,
        bytes calldata pubkey,
        bytes calldata signature
    ) external view returns (bool success);
}

/// @dev PQMLDSAI precompile interface instance at the well-known address.
PQMLDSAI constant PQMLDSA_CONTRACT = PQMLDSAI(PQMLDSA_PRECOMPILE_ADDRESS);
