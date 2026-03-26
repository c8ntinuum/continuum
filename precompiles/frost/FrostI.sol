// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev FROST verification precompile address (Cosmos EVM module constant).
address constant FROST_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000709;

/**
 * @title FrostI
 * @notice Interface for the Cosmos EVM precompile that verifies FROST signatures
 *         and signature shares using the bytemare libraries.
 *
 * @dev 1) Function names and argument ordering MUST match the precompile’s ABI:
 *        - frostVerifySignature(uint8,bytes,bytes,bytes) -> (bool)
 *        - frostVerifySignatureShare(uint8,uint16,uint16,bytes,bytes[],bytes,bytes,bytes) -> (bool)
 *
 *      2) Encodings:
 *        - `ciphersuite` is the byte id defined by bytemare/frost’s Ciphersuite enum.
 *        - `signature` is the bytemare `frost.Signature` binary encoding (NOT split R/Z).
 *        - `verificationKey` is the bytemare `ecc.Element` encoding for the suite.
 *        - `publicKeyShares[i]` are `keys.PublicKeyShare` encodings (same suite).
 *        - `commitments` is the serialized list consumed by `frost.DecodeList()`
 *           (round-1 commitments expected by the share verification).
 *        - `signatureShare` is the bytemare `frost.SignatureShare` encoding.
 *
 *      3) Execution behavior (from the Go precompile):
 *        - On ABI/type-assertion mismatch for already-unpacked parameters, the precompile
 *          returns (bool=false) rather than reverting.
 *        - On malformed ABI (e.g., can’t unpack) or decode/verify errors, it REVERTS.
 *          Callers may prefer try/catch to distinguish revert vs. false.
 *
 *      4) Gas: the precompile currently charges a constant base gas; it does not vary
 *         with input sizes (subject to change in future versions).
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
     *      This function performs no roster checks—pure signature verification
     *      against the provided group verification key.
     */
    function frostVerifySignature(
        uint8 ciphersuite,
        bytes calldata message,
        bytes calldata signature,
        bytes calldata verificationKey
    ) external pure returns (bool ok);

    /**
     * @notice Verify a single FROST signature share against a provided configuration.
     *
     * @param ciphersuite        Ciphersuite id (bytemare/frost enum, as a single byte).
     * @param threshold          Signing threshold t.
     * @param maxSigners         Maximum number of participants n.
     * @param verificationKey    bytemare-encoded `ecc.Element` group verification key.
     * @param publicKeyShares    Each signer’s bytemare-encoded `keys.PublicKeyShare`.
     * @param commitments        Serialized commitments (as expected by `frost.DecodeList()`).
     * @param message            Arbitrary message bytes (suite hashing handled inside).
     * @param signatureShare     bytemare-encoded `frost.SignatureShare`.
     * @return ok                True if the share verifies under the configuration;
     *                           may revert on decode/verify error.
     *
     * @dev Matches Go:
     *      frostVerifySignatureShare(
     *          uint8, uint16, uint16, bytes, bytes[], bytes, bytes, bytes
     *      ) returns (bool).
     *
     *      The configuration is constructed internally as:
     *        Ciphersuite=ciphersuite,
     *        Threshold=threshold,
     *        MaxSigners=maxSigners,
     *        VerificationKey=verificationKey,
     *        SignerPublicKeyShares=publicKeyShares.
     *
     *      No additional validation of `threshold/maxSigners` ranges is performed here.
     */
    function frostVerifySignatureShare(
        uint8 ciphersuite,
        uint16 threshold,
        uint16 maxSigners,
        bytes calldata verificationKey,
        bytes[] calldata publicKeyShares,
        bytes calldata commitments,
        bytes calldata message,
        bytes calldata signatureShare
    ) external pure returns (bool ok);
}

/// @dev Convenience typed handle to the precompile.
FrostI constant FROST_CONTRACT = FrostI(FROST_PRECOMPILE_ADDRESS);
