// SPDX-License-Identifier: MIT
//
// Copyright (C) 2024 Daniel Bourdrez. All Rights Reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree or at
// https://spdx.org/licenses/MIT.html

package tests_test

import (
	"bytes"
	"crypto"
	"errors"
	"testing"

	"github.com/cosmos/evm/precompiles/frost/bytemare-stable/hash"
)

type data struct {
	message []byte
	secret  []byte
	key     map[int]string
	salt    []byte
	info    []byte
}

var testData = &data{
	message: []byte("This is the message."),
	secret:  []byte("secret"),
	key: map[int]string{
		32: "2bb80d537b1da3e38bd30361aa855686bde0eacd7162fef6a25fe97bf527a25b",
		64: "bd2b1aaf7ef4f09be9f52ce2d8d599674d81aa9d6a4421696dc4d93dd0619d682ce56b4d64a9ef097761ced99e0f67265b5f76085e5b0ee7ca4696b2ad6fe2b2",
	},
	salt: nil,
	info: []byte("contextInfo"),
}

func TestID(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.HashID != h.HashID.New().Algorithm() {
			t.Error(fmtExpectedEquality, h.HashID)
		}
	})
}

func TestAvailability(t *testing.T) {
	testAll(t, func(h *testHash) {
		if !h.HashID.Available() {
			t.Errorf("%v is not available, but should be", h.HashID)
		}
	})
}

func TestNonAvailability(t *testing.T) {
	wrong := hash.Hash(crypto.MD4)
	if wrong.Available() {
		t.Errorf("%v is considered available when it should not", wrong)
	}
}

func TestFromCrypto(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.HashType == hash.FixedOutputLength {
			if hash.FromCrypto(h.cryptoID) != h.HashID {
				t.Error(fmtExpectedEquality, h.HashID)
			}
		}
	})

	if hash.FromCrypto(crypto.MD4) != 0 {
		t.Error("expected 0")
	}
}

func TestNames(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.name != h.HashID.String() {
			t.Error(fmtExpectedEquality, h.HashID)
		}
	})
}

func TestHashType(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.HashType != h.HashID.Type() {
			t.Errorf(fmtExpectedEquality, h.HashID)
		}
	})
}

func TestNoHashType(t *testing.T) {
	values := []hash.Hash{0, 20, 25, 50}
	for _, wrongID := range values {
		if wrongID.Type() != "" {
			t.Error("expected empty string")
		}
	}
}

func TestBlockSize(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.blocksize != h.HashID.New().BlockSize() {
			t.Errorf(
				"expected equality: %d:%d / %d:%d / ",
				h.HashID,
				h.blocksize,
				h.HashID.New().Algorithm(),
				h.HashID.New().BlockSize(),
			)
		}
	})
}

func TestOutputSize(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.outputsize != h.HashID.Size() || h.outputsize != h.HashID.New().Size() {
			t.Errorf(fmtExpectedEquality, h.HashID)
		}
	})
}

func TestSecurityLevel(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.security != h.HashID.SecurityLevel() {
			t.Errorf(fmtExpectedEquality, h.HashID)
		}
	})
}

func TestHashFunctions(t *testing.T) {
	testAll(t, func(h *testHash) {
		switch h.HashType {
		case hash.FixedOutputLength:
			if f := h.HashID.GetHashFunction(); f == nil {
				t.Errorf("expected pointer to be non-nil")
			}

			if f := h.HashID.GetXOF(); f != nil {
				t.Errorf("expected pointer to be nil")
			}
		case hash.ExtendableOutputFunction:
			if f := h.HashID.GetHashFunction(); f != nil {
				t.Errorf("expected pointer to be nil")
			}

			if f := h.HashID.GetXOF(); f == nil {
				t.Errorf("expected pointer to be non-nil")
			}
		default:
			panic(nil)
		}
	})
}

func TestHash(t *testing.T) {
	testAll(t, func(h *testHash) {
		hasher := h.HashID.New()
		var hashed1, hashed2 []byte

		switch h.HashType {
		case hash.FixedOutputLength:
			hashed1 = hasher.Hash(0, testData.message)
		case hash.ExtendableOutputFunction:
			hashed1 = hasher.Hash(uint(hasher.Size()), testData.message)
		}

		hashed2 = h.HashID.Hash(testData.message)

		if bytes.Compare(hashed1, hashed2) != 0 {
			t.Error(fmtExpectedEquality, h.HashID)
		}

		if len(hashed1) != h.HashID.Size() {
			t.Errorf(
				"%v : invalid hash output length length. Expected %d, got %d",
				h.HashID,
				h.HashID.Size(),
				len(hashed1),
			)
		}
	})
}

func TestSum(t *testing.T) {
	testAll(t, func(h *testHash) {
		hasher := h.HashID.New()

		_, _ = hasher.Write(testData.message)
		_, _ = hasher.Write(testData.salt)
		hashed := hasher.Sum(nil)

		if len(hashed) != hasher.Size() {
			t.Error(fmtExpectedEquality, h.HashID)
		}
	})
}

func TestRead(t *testing.T) {
	size := 100
	testAll(t, func(h *testHash) {
		hasher := h.HashID.New()

		_, _ = hasher.Write(testData.message)
		_, _ = hasher.Write(testData.salt)
		hashed1 := hasher.Read(size)
		hashed2 := hasher.Read(size)

		switch h.HashType {
		case hash.FixedOutputLength:
			if bytes.Compare(hashed1, hashed2) != 0 {
				t.Errorf(fmtExpectedEquality, h.HashID)
			}

			if len(hashed1) != h.HashID.Size() {
				t.Errorf(fmtExpectedEquality, h.HashID)
			}
		case hash.ExtendableOutputFunction:
			if bytes.Compare(hashed1, hashed2) == 0 {
				t.Errorf("%s: unexpected equality", h.HashID)
			}

			if len(hashed1) != size {
				t.Errorf(fmtExpectedEquality, h.HashID)
			}
		}
	})
}

func TestReadXOFSmallSize(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.HashType == hash.ExtendableOutputFunction {
			hasher := h.HashID.New()

			if panics, err := expectPanic(errors.New("requested output size too small"), func() {
				_ = hasher.Read(1)
			}); !panics {
				t.Errorf("expected panic: %v", err)
			}
		}
	})
}
