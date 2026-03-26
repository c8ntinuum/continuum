package types

func DefaultGenesisState() *GenesisState {
	return &GenesisState{Params: DefaultParams()}
}

func (g GenesisState) Validate() error {
	return g.Params.Validate()
}
