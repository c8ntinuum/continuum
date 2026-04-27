package ecc

import (
	"errors"
	"strings"
)

func normalizeDecodeErr(err error) error {
	if err == nil {
		return nil
	}

	msg := strings.TrimPrefix(err.Error(), "ristretto255: ")
	if msg == "invalid element encoding" {
		msg = "invalid Ristretto encoding"
	}

	return errors.New(msg)
}
