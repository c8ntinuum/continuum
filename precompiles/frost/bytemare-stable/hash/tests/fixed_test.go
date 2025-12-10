// SPDX-License-Identifier: MIT
//
// Copyright (C) 2024 Daniel Bourdrez. All Rights Reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree or at
// https://spdx.org/licenses/MIT.html

package tests_test

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/cosmos/evm/precompiles/frost/bytemare-stable/hash"
)

var errHmacKeySize = errors.New("hmac key length is larger than hash output size")

func TestHmac(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.HashType == hash.FixedOutputLength {
			hasher := h.HashID.GetHashFunction()

			key, _ := hex.DecodeString(testData.key[h.HashID.Size()])
			hmac := hasher.Hmac(testData.message, key)

			if len(hmac) != h.HashID.Size() {
				t.Errorf("#%v : invalid hmac length", h.HashID)
			}
		}
	})
}

func TestLongHmacKey(t *testing.T) {
	longHMACKey := []byte("Length65aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	testAll(t, func(h *testHash) {
		if h.HashType == hash.FixedOutputLength {
			hasher := h.HashID.GetHashFunction()

			if panics, err := expectPanic(errHmacKeySize, func() {
				_ = hasher.Hmac(testData.message, longHMACKey)
			}); !panics {
				t.Errorf("expected panic: %v", err)
			}
		}
	})
}

func TestHKDF(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.HashType == hash.FixedOutputLength {
			hasher := h.HashID.GetHashFunction()

			for _, l := range []int{0, h.HashID.Size()} {
				key := hasher.HKDF(testData.secret, testData.salt, testData.info, l)

				if len(key) != h.HashID.Size() {
					t.Errorf("#%v : invalid key length (length argument = %d)", h.HashID, l)
				}
			}
		}
	})
}

func TestHKDFExtract(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.HashType == hash.FixedOutputLength {
			hasher := h.HashID.GetHashFunction()

			for _, l := range []int{0, h.HashID.Size()} {
				// Build a pseudorandom key
				prk := hasher.HKDFExtract(testData.secret, testData.salt)

				if len(prk) != h.HashID.Size() {
					t.Errorf("%v : invalid key length (length argument = %d)", h.HashID, l)
				}
			}
		}
	})
}

func TestHKDFExpand(t *testing.T) {
	testAll(t, func(h *testHash) {
		if h.HashType == hash.FixedOutputLength {
			hasher := h.HashID.GetHashFunction()

			for _, l := range []int{0, h.HashID.Size()} {
				// Build a pseudorandom key
				prk := hasher.HKDF(testData.secret, testData.salt, testData.info, l)
				key := hasher.HKDFExpand(prk, testData.info, l)

				if len(key) != h.HashID.Size() {
					t.Errorf("#%v : invalid key length (length argument = %d)", h.HashID, l)
				}
			}
		}
	})
}
