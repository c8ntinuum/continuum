// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev FROST verification precompile address (Cosmos EVM module constant).
address constant FROST_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000709;

/**
 * @title FrostI
 * @notice Interface for the Cosmos EVM precompile that verifies caller-supplied
 *         FROST signatures using the bytemare libraries.
 *
 * @dev 1) Function names and argument ordering MUST match the precompile’s ABI:
 *        - frostVerifySignature(uint8,bytes,bytes,bytes) -> (bool)
 *
 *      2) Encodings:
 *        - `ciphersuite` is the byte id defined by bytemare/frost’s Ciphersuite enum.
 *        - `signature` is the bytemare `frost.Signature` binary encoding (NOT split R/Z).
 *        - `verificationKey` is the bytemare `ecc.Element` encoding for the suite.
 *
 *      3) Execution behavior (from the Go precompile):
 *        - On ABI/type-assertion mismatch for already-unpacked parameters, the precompile
 *          returns (bool=false) rather than reverting.
 *        - On malformed ABI (e.g., can’t unpack) or decode/verify errors, it REVERTS.
 *          Callers may prefer try/catch to distinguish revert vs. false.
 *
 *      4) Security model:
 *        - This precompile only verifies a signature against the caller-provided
 *          verification key.
 *        - It does NOT prove endorsement by any pre-authorized threshold group.
 *        - Any roster, threshold, or group-commitment policy must be enforced by
 *          the caller at a higher layer.
 *
 *      5) Gas: the precompile charges a fixed base plus per-word calldata cost.
 */
interface FrostI {
    /**
     * @notice Verify a FROST signature (single-signer or already aggregated).
     *
     * @param ciphersuite      Ciphersuite id (bytemare/frost enum, as a single byte).
     * @param message          Arbitrary message bytes (the precompile handles suite hashing).
     * @param signature        bytemare-encoded `frost.Signature` (NOT split into R/Z).
     * @param verificationKey  bytemare-encoded `ecc.Element` public key for the suite.
     * @return ok              True if the signature verifies; may revert on decode/verify error.
     *
     * @dev Matches Go: frostVerifySignature(uint8,bytes,bytes,bytes) returns (bool).
     *      This function performs no roster checks. It is pure signature verification
     *      against the provided verification key.
     */
    function frostVerifySignature(
        uint8 ciphersuite,
        bytes calldata message,
        bytes calldata signature,
        bytes calldata verificationKey
    ) external pure returns (bool ok);
}

/// @dev Convenience typed handle to the precompile.
FrostI constant FROST_CONTRACT = FrostI(FROST_PRECOMPILE_ADDRESS);
