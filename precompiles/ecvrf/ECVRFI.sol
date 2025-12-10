// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev ECVRF verification precompile address.
address constant ECVRF_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000708;

/// @notice Interface for verifying ECVRF proofs (VeChain go-ecvrf).
/// @dev Supported suites (case-insensitive): "P256_SHA256_TAI", "SECP256K1_SHA256_TAI".
interface ECVRFI {
    /**
     * @notice Verify an ECVRF proof and return the VRF output (beta).
     * @dev
     *  - `pubKey` is the ECDSA public key in SEC1 form:
     *      * uncompressed (65 bytes, 0x04 || X || Y), or
     *      * compressed   (33 bytes, 0x02/0x03 || X).
     *  - `pi` is the VRF proof produced by go-ecvrf.
     *  - `alpha` is the input message bytes.
     * @param suite   "P256_SHA256_TAI" or "SECP256K1_SHA256_TAI"
     * @param pubKey  Public key (33 or 65 bytes, SEC1)
     * @param alpha   Message input
     * @param pi      VRF proof
     * @return ok     True iff the proof verifies
     * @return beta   VRF hash output (empty if `ok` is false)
     */
    function ecvrfVerify(
        string calldata suite,
        bytes calldata pubKey,
        bytes calldata alpha,
        bytes calldata pi
    ) external pure returns (bool ok, bytes memory beta);
}

ECVRFI constant ECVRF_CONTRACT = ECVRFI(ECVRF_PRECOMPILE_ADDRESS);
