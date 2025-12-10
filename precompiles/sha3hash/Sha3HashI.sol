// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev SHA3 (FIPS-202) hashing precompile address.
address constant SHA3_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000706;

/// @notice Compute SHA3-256 or SHA3-512 over arbitrary bytes.
/// @dev `hashName` is case-insensitive; accepted values:
///      "SHA3-256", "SHA3_256", "SHA3-512", "SHA3_512".
///      Output length is 32 bytes for SHA3-256, 64 bytes for SHA3-512.
interface Sha3HashI {
    /**
     * @param data     Arbitrary input bytes to hash.
     * @param hashName Algorithm selector ("SHA3-256" or "SHA3-512").
     * @return digest  Raw digest bytes.
     */
    function sha3Hash(
        bytes calldata data,
        string calldata hashName
    ) external pure returns (bytes memory digest);
}

Sha3HashI constant SHA3_HASH_CONTRACT = Sha3HashI(SHA3_PRECOMPILE_ADDRESS);
