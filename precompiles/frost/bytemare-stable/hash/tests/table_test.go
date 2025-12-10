// SPDX-License-Identifier: MIT
//
// Copyright (C) 2024 Daniel Bourdrez. All Rights Reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree or at
// https://spdx.org/licenses/MIT.html

package tests_test

import (
	"crypto"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"testing"

	"github.com/cosmos/evm/precompiles/frost/bytemare-stable/hash"
)

const (
	// string IDs for the hash functions.
	shake128 = "SHAKE128"
	shake256 = "SHAKE256"
	blake2xb = "BLAKE2XB"
	blake2xs = "BLAKE2XS"
)

type testHash struct {
	HashType   hash.Type
	name       string
	cryptoID   crypto.Hash
	blocksize  int
	outputsize int
	security   int
	HashID     hash.Hash
}

const (
	blockSHA3256 = 1088 / 8
	blockSHA3384 = 832 / 8
	blockSHA3512 = 576 / 8
)

var testHashes = []*testHash{
	{hash.FixedOutputLength, crypto.SHA256.String(), crypto.SHA256, sha256.BlockSize, sha256.Size, 128, hash.SHA256},
	{hash.FixedOutputLength, crypto.SHA384.String(), crypto.SHA384, sha512.BlockSize, sha512.Size384, 192, hash.SHA384},
	{hash.FixedOutputLength, crypto.SHA512.String(), crypto.SHA512, sha512.BlockSize, sha512.Size, 256, hash.SHA512},
	{
		hash.FixedOutputLength,
		crypto.SHA3_256.String(),
		crypto.SHA3_256,
		blockSHA3256,
		crypto.SHA3_256.Size(),
		128,
		hash.SHA3_256,
	},
	{
		hash.FixedOutputLength,
		crypto.SHA3_384.String(),
		crypto.SHA3_384,
		blockSHA3384,
		crypto.SHA3_384.Size(),
		192,
		hash.SHA3_384,
	},
	{
		hash.FixedOutputLength,
		crypto.SHA3_512.String(),
		crypto.SHA3_512,
		blockSHA3512,
		crypto.SHA3_512.Size(),
		256,
		hash.SHA3_512,
	},
	{hash.ExtendableOutputFunction, shake128, crypto.Hash(0), 168, 32, 128, hash.SHAKE128},
	{hash.ExtendableOutputFunction, shake256, crypto.Hash(0), 136, 32, 224, hash.SHAKE256},
	{hash.ExtendableOutputFunction, blake2xb, crypto.Hash(0), 0, 32, 128, hash.BLAKE2XB},
	{hash.ExtendableOutputFunction, blake2xs, crypto.Hash(0), 0, 32, 128, hash.BLAKE2XS},
}

func testAll(t *testing.T, f func(*testHash)) {
	for _, test := range testHashes {
		t.Run(test.name, func(t *testing.T) {
			f(test)
		})
	}
}

var (
	errNoPanic          = errors.New("no panic")
	errNoPanicMessage   = errors.New("panic but no message")
	fmtExpectedEquality = "%s: expected equality"
)

func hasPanic(f func()) (has bool, err error) {
	err = nil
	var report interface{}
	func() {
		defer func() {
			if report = recover(); report != nil {
				has = true
			}
		}()

		f()
	}()

	if has {
		err = fmt.Errorf("%v", report)
	}

	return has, err
}

// expectPanic executes the function f with the expectation to recover from a panic. If no panic occurred or if the
// panic message is not the one expected, ExpectPanic returns (false, error).
func expectPanic(expectedError error, f func()) (bool, error) {
	hasPanic, err := hasPanic(f)

	if !hasPanic {
		return false, errNoPanic
	}

	if expectedError == nil {
		return true, nil
	}

	if err == nil {
		return false, errNoPanicMessage
	}

	if err.Error() != expectedError.Error() {
		return false, fmt.Errorf("expected %q, got: %w", expectedError, err)
	}

	return true, nil
}
