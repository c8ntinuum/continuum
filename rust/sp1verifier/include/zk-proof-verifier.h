#include <cstdarg>
#include <cstdint>
#include <cstdlib>
#include <ostream>
#include <new>

extern "C" {

/// C interface wrapper
int verify_groth16_c(const unsigned char *proof_ptr,
                     uintptr_t proof_len,
                     const unsigned char *inputs_ptr,
                     uintptr_t inputs_len,
                     const char *hash_ptr);

/// C interface wrapper
int verify_plonk_c(const unsigned char *proof_ptr,
                   uintptr_t proof_len,
                   const unsigned char *inputs_ptr,
                   uintptr_t inputs_len,
                   const char *hash_ptr);

}  // extern "C"
