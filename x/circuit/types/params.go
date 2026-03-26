package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultParams() Params {
	return Params{
		Whitelist: []string{},
	}
}

func (p Params) Validate() error {
	seen := make(map[string]struct{}, len(p.Whitelist))
	for _, addr := range p.Whitelist {
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return fmt.Errorf("invalid whitelist address: %w", err)
		}
		if _, ok := seen[addr]; ok {
			return fmt.Errorf("duplicate whitelist address: %s", addr)
		}
		seen[addr] = struct{}{}
	}
	return nil
}
