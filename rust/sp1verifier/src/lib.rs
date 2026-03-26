use sp1_verifier::Groth16Error;
use sp1_verifier::Groth16Verifier;
use sp1_verifier::PlonkError;
use sp1_verifier::PlonkVerifier;
use std::ffi::{c_char, CStr};
use std::os::raw::{c_int, c_uchar};

pub fn verify_plonk(
    proof: &[u8],
    sp1_public_inputs: &[u8],
    sp1_vkey_hash: &str,
) -> Result<(), PlonkError> {
    let plonk_vk = *sp1_verifier::PLONK_VK_BYTES;
    PlonkVerifier::verify(proof, sp1_public_inputs, sp1_vkey_hash, plonk_vk)
}

pub fn verify_groth16(
    proof: &[u8],
    sp1_public_inputs: &[u8],
    sp1_vkey_hash: &str,
) -> Result<(), Groth16Error> {
    let groth16_vk = *sp1_verifier::GROTH16_VK_BYTES;
    Groth16Verifier::verify(proof, sp1_public_inputs, sp1_vkey_hash, groth16_vk)
}

/// C interface wrapper
#[no_mangle]
pub extern "C" fn verify_groth16_c(
    proof_ptr: *const c_uchar,
    proof_len: usize,
    inputs_ptr: *const c_uchar,
    inputs_len: usize,
    hash_ptr: *const c_char,
) -> c_int {
    unsafe {
        if proof_ptr.is_null() || inputs_ptr.is_null() || hash_ptr.is_null() {
            return 1;
        }
        if proof_len == 0 || inputs_len == 0 {
            return 1;
        }
        let proof: &[u8] = std::slice::from_raw_parts(proof_ptr, proof_len);
        let inputs: &[u8] = std::slice::from_raw_parts(inputs_ptr, inputs_len);
        let hash: &str = match CStr::from_ptr(hash_ptr).to_str() {
            Ok(h) => h,
            Err(_) => return 2,
        };
        match verify_groth16(proof, inputs, hash) {
            Ok(_) => 0,
            Err(Groth16Error::ProofVerificationFailed) => 3,
            Err(Groth16Error::Groth16VkeyHashMismatch) => 4,
            Err(_) => 5,
        }
    }
}

/// C interface wrapper
#[no_mangle]
pub extern "C" fn verify_plonk_c(
    proof_ptr: *const c_uchar,
    proof_len: usize,
    inputs_ptr: *const c_uchar,
    inputs_len: usize,
    hash_ptr: *const c_char,
) -> c_int {
    unsafe {
        if proof_ptr.is_null() || inputs_ptr.is_null() || hash_ptr.is_null() {
            return 1;
        }
        if proof_len == 0 || inputs_len == 0 {
            return 1;
        }
        let proof: &[u8] = std::slice::from_raw_parts(proof_ptr, proof_len);
        let inputs: &[u8] = std::slice::from_raw_parts(inputs_ptr, inputs_len);
        let hash: &str = match CStr::from_ptr(hash_ptr).to_str() {
            Ok(h) => h,
            Err(_) => return 2,
        };
        match verify_plonk(proof, inputs, hash) {
            Ok(_) => 0,
            Err(PlonkError::PairingCheckFailed) => 3,
            Err(PlonkError::PlonkVkeyHashMismatch) => 4,
            Err(_) => 5,
        }
    }
}
