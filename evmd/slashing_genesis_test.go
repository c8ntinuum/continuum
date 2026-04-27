package evmd

import (
	"encoding/json"
	"testing"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/config"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func TestInitChainerSeedsMissingSlashingSigningInfos(t *testing.T) {
	chainID := "slashing-signing-info-seed-test"

	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)

	validator := cmttypes.NewValidator(pubKey, 1)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})

	depositorPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(
		depositorPrivKey.PubKey().Address().Bytes(),
		depositorPrivKey.PubKey(),
		0,
		0,
	)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins: sdk.NewCoins(
			sdk.NewCoin(evmtypes.DefaultEVMExtendedDenom, sdkmath.NewInt(1_000_000)),
		),
	}

	app, genesisState := setup(true, 5, chainID, config.EVMChainID)
	genesisState, err = simtestutil.GenesisStateWithValSet(app.AppCodec(), genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)
	require.NoError(t, err)

	var bankGenesis banktypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGenesis)
	bankGenesis.DenomMetadata = []banktypes.Metadata{defaultDenomMetadata(evmtypes.DefaultEVMExtendedDenom)}
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(&bankGenesis)

	var slashingGenesis slashingtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesisState[slashingtypes.ModuleName], &slashingGenesis)
	slashingGenesis.SigningInfos = nil
	slashingGenesis.MissedBlocks = nil
	genesisState[slashingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(&slashingGenesis)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	require.NoError(t, err)

	_, err = app.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: simtestutil.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
		ChainId:         chainID,
	})
	require.NoError(t, err)

	ctx := app.NewContextLegacy(false, tmproto.Header{
		ChainID: chainID,
		Height:  1,
	})

	signingInfo, err := app.SlashingKeeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(validator.Address.Bytes()))
	require.NoError(t, err)
	require.Equal(t, int64(0), signingInfo.StartHeight)
	require.Zero(t, signingInfo.IndexOffset)
	require.Zero(t, signingInfo.MissedBlocksCounter)
}
