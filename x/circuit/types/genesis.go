package types

import "fmt"

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		State: CircuitState{
			SystemAvailable: true,
		},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	if !gs.State.SystemAvailable && len(gs.Params.Whitelist) == 0 {
		return fmt.Errorf("whitelist cannot be empty while system_available is false")
	}
	return nil
}
