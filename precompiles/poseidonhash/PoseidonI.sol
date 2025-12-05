// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev Poseidon hash precompile address (3-input, BN254 field).
address constant POSEIDON_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000711;

/// @dev Poseidon precompile interface bound to the well-known address.
/// @notice Computes Poseidon hash over 3 field elements (BN254 scalar field).
///         The implementation is expected to match github.com/iden3/go-iden3-crypto/poseidon
///         for 3 inputs (the variant used in your Go tests).
interface PoseidonI {
    /**
     * @notice Computes the Poseidon hash of 3 field elements.
     * @dev Each input is interpreted as an element of the BN254 scalar field
     *      (i.e. reduced modulo r).
     * @param inputs  Array of 3 uint256 field elements.
     * @return hash   Poseidon hash output as a uint256 field element.
     */
    function poseidonHash(
        uint256[3] calldata inputs
    ) external pure returns (uint256 hash);
}

/// @dev Poseidon precompile interface instance at the well-known address.
PoseidonI constant POSEIDON_CONTRACT = PoseidonI(POSEIDON_PRECOMPILE_ADDRESS);
