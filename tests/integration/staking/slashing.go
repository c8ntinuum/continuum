package staking

import (
	"testing"

	"github.com/cosmos/evm/testutil/integration/evm/network"

	sdkmath "cosmossdk.io/math"
	evidencetypes "cosmossdk.io/x/evidence/types"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

func TestSlashingSigningInfoIntegration(t *testing.T, create network.CreateEvmApp, options ...network.ConfigOption) {
	t.Run("MsgCreateValidator seeds signing info", func(t *testing.T) {
		nw, tf, kr := setupNetwork(t, create, options...)

		minSelfDelegation := mustInt(t, minSelfDelegationStr)
		selfDelegation := sdk.NewCoin(nw.GetBaseDenom(), minSelfDelegation)
		consKey := ed25519.GenPrivKey().PubKey()
		msg := buildCreateValidatorMsgWithConsKey(t, kr.GetPrivKey(0), consKey, selfDelegation, minSelfDelegation)

		res, err := commitCosmosTx(tf, kr.GetPrivKey(0), msg)
		require.NoError(t, err)
		require.Zerof(t, res.Code, "log: %s", res.Log)

		// This assertion is the real post-genesis validator path:
		// tx delivery -> staking state transition -> staking hooks -> slashing row.
		ctx := nw.GetContext()
		consAddr := sdk.ConsAddress(consKey.Address())
		info, err := nw.App.GetSlashingKeeper().GetValidatorSigningInfo(ctx, consAddr)
		require.NoError(t, err, "MsgCreateValidator should create signing info through the real staking lifecycle")
		require.Equal(t, ctx.BlockHeight(), info.StartHeight, "StartHeight should come from the block that bonded the validator")
		require.Zero(t, info.IndexOffset, "newly created validators should start with a clean signing window")
		require.Zero(t, info.MissedBlocksCounter, "newly created validators should start with no missed blocks recorded")
	})

	t.Run("Equivocation evidence does not panic when genesis signing info was missing", func(t *testing.T) {
		slashingGen := slashingtypes.DefaultGenesisState()
		slashingGen.SigningInfos = nil
		slashingGen.MissedBlocks = nil

		opts := append([]network.ConfigOption{
			network.WithAmountOfValidators(1),
			network.WithCustomGenesis(network.CustomGenesisState{
				slashingtypes.ModuleName: slashingGen,
			}),
		}, options...)

		nw, _, _ := setupNetwork(t, create, opts...)
		ctx := nw.GetContext()
		genesisValidator := nw.GetValidators()[0]
		consAddrBz, err := genesisValidator.GetConsAddr()
		require.NoError(t, err)
		consAddr := sdk.ConsAddress(consAddrBz)

		_, err = nw.App.GetSlashingKeeper().GetValidatorSigningInfo(ctx, consAddr)
		require.NoError(t, err, "InitChainer should seed missing signing info before evidence handling")

		// Drive duplicate-vote handling through FinalizeBlock misbehavior so this
		// test covers the actual block/evidence path, not just slashing keeper primitives.
		res, err := nw.NextBlockWithMisbehavior(abcitypes.Misbehavior{
			Type:   abcitypes.MisbehaviorType_DUPLICATE_VOTE,
			Height: ctx.BlockHeight(),
			Time:   ctx.BlockTime(),
			Validator: abcitypes.Validator{
				Address: consAddr,
				Power:   1,
			},
			TotalVotingPower: 1,
		})
		require.NoError(t, err)
		require.NotNil(t, res)

		postCtx := nw.GetContext()
		jailed, err := nw.App.GetStakingKeeper().IsValidatorJailed(postCtx, consAddr)
		require.NoError(t, err)
		require.True(t, jailed, "duplicate-vote evidence should jail the validator through the evidence path")

		info, err := nw.App.GetSlashingKeeper().GetValidatorSigningInfo(postCtx, consAddr)
		require.NoError(t, err)
		require.True(t, info.Tombstoned, "duplicate-vote evidence should tombstone the validator")
		require.Equal(t, evidencetypes.DoubleSignJailEndTime.UTC(), info.JailedUntil.UTC(), "double-sign evidence should apply the permanent jail end time")
	})
}

func buildCreateValidatorMsgWithConsKey(
	t *testing.T,
	priv cryptotypes.PrivKey,
	consKey cryptotypes.PubKey,
	selfDelegation sdk.Coin,
	minSelfDelegation sdkmath.Int,
) *stakingtypes.MsgCreateValidator {
	t.Helper()

	valAddr := sdk.ValAddress(priv.PubKey().Address()).String()
	desc := stakingtypes.NewDescription("moniker", "identity", "", "", "")
	commission := stakingtypes.NewCommissionRates(
		sdkmath.LegacyNewDecWithPrec(1, 1),
		sdkmath.LegacyNewDecWithPrec(1, 1),
		sdkmath.LegacyNewDecWithPrec(1, 1),
	)

	msg, err := stakingtypes.NewMsgCreateValidator(
		valAddr,
		consKey,
		selfDelegation,
		desc,
		commission,
		minSelfDelegation,
	)
	require.NoError(t, err)
	return msg
}
