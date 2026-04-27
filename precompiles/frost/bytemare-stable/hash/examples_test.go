// SPDX-License-Identifier: MIT
//
// Copyright (C) 2024 Daniel Bourdrez. All Rights Reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree or at
// https://spdx.org/licenses/MIT.html

package hash_test

import (
	"crypto"
	"encoding/hex"
	"fmt"

	"github.com/cosmos/evm/precompiles/frost/bytemare-stable/hash"
)

// Example_Hashing shows how to hash an input.
func Example_hashing() {
	input := []byte("input")

	output := hash.SHA256.Hash(input)
	fmt.Printf("%s of %q = %s\n", hash.SHA256, input, hex.EncodeToString(output))

	// Output: SHA-256 of "input" = c96c6d5be8d08a12e7b5cdc1b207fa6b2430974c86803d8891675e76fd992c20
}

// Example_Hmac shows how to compute a HMAC for a message and key.
func Example_hmac() {
	message := []byte("message")
	key := []byte("key")
	h := hash.SHA256

	hmac := h.GetHashFunction().Hmac(message, key)
	fmt.Printf("HMAC(%s) of (%s,%s) = %s\n", h, message, key, hex.EncodeToString(hmac))

	// Output: HMAC(SHA-256) of (message,key) = 6e9ef29b75fffc5b7abae527d58fdadb2fe42e7219011976917343065f58ed4a
}

// Example_HKDF shows how to derive a key with HKDF from an input secret, salt, and additional info.
func Example_hkdf() {
	secret := []byte("secret")
	salt := []byte("salt")
	info := []byte("info")
	outputLength := 32
	h := hash.SHA256

	hkdf := h.GetHashFunction().HKDF(secret, salt, info, outputLength)
	fmt.Printf(
		"HKDF(%s) of (%s,%s,%s) for %d bytes = %s\n",
		h,
		secret,
		salt,
		info,
		outputLength,
		hex.EncodeToString(hkdf),
	)

	// Output: HKDF(SHA-256) of (secret,salt,info) for 32 bytes = f6d2fcc47cb939deafe3853a1e641a27e6924aff7a63d09cb04ccfffbe4776ef
}

// Example_HKDF_Extract_Expand shows how to derive multiple keys with HKDF-Extract-and-Expand from an input secret, salt,
// and additional info for each key.
func Example_hkdf_extract_expand() {
	secret := []byte("secret")
	salt := []byte("salt")
	keyInfo := []string{"key1", "key2", "key3", "key4"}
	outputLength := 32
	h := hash.SHA256.GetHashFunction()

	prk := h.HKDFExtract(secret, salt)

	fmt.Printf(
		"HKDF-Expanded output keys from extracted pseudorandom key %q on %d bytes\n",
		hex.EncodeToString(prk),
		outputLength,
	)
	for _, info := range keyInfo {
		key := h.HKDFExpand(prk, []byte(info), outputLength)
		fmt.Printf("%s = %s\n", info, hex.EncodeToString(key))
	}

	// Output: HKDF-Expanded output keys from extracted pseudorandom key "98e5340f0f4f96d2b80c2a90da0d03cf46c35e9492918cc7af73d9a39efa5981" on 32 bytes
	// key1 = f490601be934fe13381586ba657fae4534c0921345d41b97b804bf76ba29664b
	// key2 = cca6ff4021287207e49c5e8297bea41b405eed697f78ef1174707a0bfcf70da7
	// key3 = a9d11fb5ce71802b6a4c19e7bb45c51aa7e131ea3b673e1fb77a6698babbf1ea
	// key4 = ff0330c4aaf9cc58db65a5346b0e97050856649e2cc0a256038133c30b420bfc
}

// Example_Crypto shows how to use this package if you already have a hash identifier from the built-in crypto package.
func Example_crypto() {
	input := []byte("input")

	output := hash.FromCrypto(crypto.SHA256).Hash(input)
	fmt.Printf("%s of %q = %s\n", hash.FromCrypto(crypto.SHA256), input, hex.EncodeToString(output))

	// Output: SHA-256 of "input" = c96c6d5be8d08a12e7b5cdc1b207fa6b2430974c86803d8891675e76fd992c20
}

// Example_Info shows what hash function metadata is available.
func Example_info() {
	ids := []hash.Hash{hash.SHA512, hash.BLAKE2XS}

	for _, id := range ids {
		fmt.Printf("Hash function: %s\n", id)
		fmt.Printf("Is available? %v\n", id.Available())
		fmt.Printf("Hash type: %s\n", id.Type())
		fmt.Printf("Security level (bits): %d\n", id.SecurityLevel())
		fmt.Printf("Standard output size (bytes): %d\n", id.Size())
		fmt.Printf("Underlying block size (bytes): %d\n", id.BlockSize())
	}

	fmt.Printf("NOTE that the block size is only relevant for fixed output length functions, and is set to 0 for XOF")

	// Output: Hash function: SHA-512
	// Is available? true
	// Hash type: fixed
	// Security level (bits): 256
	// Standard output size (bytes): 64
	// Underlying block size (bytes): 128
	// Hash function: BLAKE2XS
	// Is available? true
	// Hash type: extendable-output-function
	// Security level (bits): 128
	// Standard output size (bytes): 32
	// Underlying block size (bytes): 0
	// NOTE that the block size is only relevant for fixed output length functions, and is set to 0 for XOF
}
