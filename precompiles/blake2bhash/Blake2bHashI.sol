// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev BLAKE2b hashing precompile address.
address constant BLAKE2B_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000707;

/// @notice Compute BLAKE2b-256 or BLAKE2b-512 over arbitrary bytes.
/// @dev `hashName` is case-insensitive; accepted values:
///      "BLAKE2B-256", "BLAKE2B_256", "BLAKE2B-512", "BLAKE2B_512".
///      Output length is 32 bytes for -256, 64 bytes for -512.
interface Blake2bHashI {
    /**
     * @param data     Arbitrary input bytes to hash.
     * @param hashName Algorithm selector ("BLAKE2B-256" or "BLAKE2B-512").
     * @return digest  Raw digest bytes.
     */
    function blake2bHash(
        bytes calldata data,
        string calldata hashName
    ) external pure returns (bytes memory digest);
}

Blake2bHashI constant BLAKE2B_HASH_CONTRACT = Blake2bHashI(
    BLAKE2B_PRECOMPILE_ADDRESS
);
