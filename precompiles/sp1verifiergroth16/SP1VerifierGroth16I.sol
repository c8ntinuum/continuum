// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The SP1Verifier Groth16 contract's address.
address constant SP1VERIFIERGROTH16_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000600;

/// @dev The SP1Verifier Groth16 contract's instance.
SP1VerifierGroth16I constant SP1VERIFIERGROTH16_CONTRACT = SP1VerifierGroth16I(
    SP1VERIFIERGROTH16_PRECOMPILE_ADDRESS
);

/// @title SP1 Groth 16 Verifier Interface
/// @author Succinct Labs
/// @notice This contract is the interface for the SP1 Verifier.
interface SP1VerifierGroth16I {
    /// @notice Verifies a proof with given public values and vkey.
    /// @dev It is expected that the first 4 bytes of proofBytes must match the first 4 bytes of
    /// target verifier's VERIFIER_HASH.
    /// @param programVKey The verification key for the RISC-V program.
    /// @param publicValues The public values encoded as bytes.
    /// @param proofBytes The proof of the program execution the SP1 zkVM encoded as bytes.
    function verifyProof(
        bytes32 programVKey,
        bytes calldata publicValues,
        bytes calldata proofBytes
    ) external view;

    /// @notice Returns the hash of the verifier.
    function VERIFIER_HASH() external pure returns (bytes32);

    function VERSION() external pure returns (string memory);
}
