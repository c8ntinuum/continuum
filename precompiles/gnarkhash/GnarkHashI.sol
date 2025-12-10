// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev Gnark-crypto hashes precompile address.
address constant GNARK_HASH_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000705;

/// @notice Compute a gnark-crypto hash over arbitrary bytes.
/// @dev `hashName` must be one of the gnark-crypto standard names
///      (e.g. "MIMC_BN254", "POSEIDON2_BN254", "POSEIDON2_BLS12_381", ...).
///      Output length depends on the hash (e.g., 32/48/96/4/8 bytes).
interface GnarkHashI {
    /**
     * @param data     Arbitrary input bytes to hash.
     * @param hashName Name of gnark-crypto hash function (case-insensitive).
     * @return digest  The raw digest bytes for the chosen function.
     */
    function gnarkHash(
        bytes calldata data,
        string calldata hashName
    ) external pure returns (bytes memory digest);
}

GnarkHashI constant GNARK_HASH_CONTRACT = GnarkHashI(
    GNARK_HASH_PRECOMPILE_ADDRESS
);
