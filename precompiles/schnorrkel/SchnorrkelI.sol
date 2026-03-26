// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev Schnorrkel (sr25519) precompile address.
address constant SCHNORRKEL_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000704;

/// @notice Verify Schnorrkel (sr25519) signatures using a signing context and message.
/// @dev Param sizes:
///      - pubKey:    32 bytes (sr25519 public key)
///      - signature: 64 bytes
///      - signingCtx: arbitrary bytes (domain separation label)
///      - msg:        arbitrary bytes (the signed message payload)
interface SchnorrkelI {
    /**
     * @param signingCtx  Domain separation context bytes (same bytes used for signing).
     * @param msg         Message bytes (same bytes used for signing).
     * @param pubKey      32-byte sr25519 public key.
     * @param signature   64-byte Schnorrkel signature.
     * @return success    true iff signature is valid for (signingCtx, msg, pubKey).
     */
    function verifySchnorrkelSignature(
        bytes calldata signingCtx,
        bytes calldata msg,
        bytes calldata pubKey,
        bytes calldata signature
    ) external pure returns (bool success);
}

SchnorrkelI constant SCHNORRKEL_CONTRACT = SchnorrkelI(
    SCHNORRKEL_PRECOMPILE_ADDRESS
);
