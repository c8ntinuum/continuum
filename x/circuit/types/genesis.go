package types

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		State: CircuitState{
			SystemAvailable: true,
		},
	}
}

func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
