package types

import (
	fmtpkg "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultParams() Params {
	return Params{Whitelist: []string{}}
}

func (p Params) Validate() error {
	seen := make(map[string]struct{}, len(p.Whitelist))
	for _, addr := range p.Whitelist {
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return fmtpkg.Errorf("invalid whitelist address %q: %w", addr, err)
		}
		if _, ok := seen[addr]; ok {
			return fmtpkg.Errorf("duplicate whitelist address %q", addr)
		}
		seen[addr] = struct{}{}
	}
	return nil
}
