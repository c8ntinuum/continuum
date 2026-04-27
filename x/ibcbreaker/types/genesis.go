package types

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		State: IbcBreakerState{
			IbcAvailable: true,
		},
	}
}

func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}
