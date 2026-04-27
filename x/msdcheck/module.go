package msdcheck

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	staking "github.com/cosmos/cosmos-sdk/x/staking"
	stakingexported "github.com/cosmos/cosmos-sdk/x/staking/exported"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type AppModule struct {
	staking.AppModule
	keeper         *stakingkeeper.Keeper
	legacySubspace stakingexported.Subspace
}

func NewAppModule(
	cdc codec.Codec,
	keeper *stakingkeeper.Keeper,
	accountKeeper stakingtypes.AccountKeeper,
	bankKeeper stakingtypes.BankKeeper,
	ls stakingexported.Subspace,
) AppModule {
	return AppModule{
		AppModule:      staking.NewAppModule(cdc, keeper, accountKeeper, bankKeeper, ls),
		keeper:         keeper,
		legacySubspace: ls,
	}
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	base := stakingkeeper.NewMsgServerImpl(am.keeper)
	wrapped := NewMsgServer(base, am.keeper)
	stakingtypes.RegisterMsgServer(cfg.MsgServer(), wrapped)

	querier := stakingkeeper.Querier{Keeper: am.keeper}
	stakingtypes.RegisterQueryServer(cfg.QueryServer(), querier)

	m := stakingkeeper.NewMigrator(am.keeper, am.legacySubspace)
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(err)
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(err)
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(err)
	}
}
