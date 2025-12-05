package msd

import (
	"encoding/hex"
	"fmt"
	"math/big"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const minSelfDelegation = "888888000000000000000000"

var (
	MinSelfDelegation = math.NewIntFromBigInt(big.NewInt(0))
	ErrMSD            = sdkerrors.Register("msd", 2, "MSD err")
	sk                *stakingkeeper.Keeper
	slk               slashingkeeper.Keeper
	bondDenom         string
	appCodec          codec.Codec
)

func Init(_stakingKeeper *stakingkeeper.Keeper, _slashingkeeper slashingkeeper.Keeper, _denom string, _codec codec.Codec) {
	i, ok := math.NewIntFromString(minSelfDelegation)
	if !ok {
		panic("bad MinSelfDelegation literal")
	}
	MinSelfDelegation = i
	sk = _stakingKeeper
	slk = _slashingkeeper
	bondDenom = _denom
	appCodec = _codec
}

func CheckMSDInTx(ctx sdk.Context, tx sdk.Tx) error {
	for _, msg := range tx.GetMsgs() {
		if err := CheckMSDInMsg(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func CheckMSDInMsg(ctx sdk.Context, msg sdk.Msg) error {
	if exec, ok := msg.(*authz.MsgExec); ok {
		for _, any := range exec.Msgs {
			var inner sdk.Msg
			if err := appCodec.UnpackAny(any, &inner); err != nil {
				return fmt.Errorf("authz unpack failed: %w", err)
			}
			if err := CheckMSDInMsg(ctx, inner); err != nil {
				return err
			}
		}
		return nil
	}

	switch m := msg.(type) {

	case *stakingtypes.MsgCreateValidator:
		err := checkCreateValidator(m)
		if err != nil {
			return err
		}

	case *stakingtypes.MsgEditValidator:
		err := checkEditValidator(m)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkCreateValidator(msg *stakingtypes.MsgCreateValidator) error {
	if msg.Value.Denom != bondDenom {
		return fmt.Errorf("initial self-delegation must be in bond denom %s", bondDenom)
	}

	if msg.Value.Amount.LT(MinSelfDelegation) {
		return fmt.Errorf("initial self-delegation amount %s < MSD %s", msg.Value.Amount.String(), MinSelfDelegation.String())
	}

	if msg.MinSelfDelegation.LT(MinSelfDelegation) {
		return fmt.Errorf("initial self-delegation specified %s < MSD %s", msg.MinSelfDelegation.String(), MinSelfDelegation.String())
	}

	return nil
}

func checkEditValidator(msg *stakingtypes.MsgEditValidator) error {
	if msg.MinSelfDelegation.LT(MinSelfDelegation) {
		return fmt.Errorf("proposed self-delegation specified %s < MSD %s", msg.MinSelfDelegation.String(), MinSelfDelegation.String())
	}
	return nil
}

func Jail(ctx sdk.Context, proposerHexAddress string) error {
	proposerBz, err := hex.DecodeString(proposerHexAddress)
	if err != nil {
		return err
	}
	consAddr := sdk.ConsAddress(proposerBz)
	return slk.Jail(ctx, consAddr)
}

func Slash(ctx sdk.Context, proposerHexAddress string, height int64) error {
	proposerBz, err := hex.DecodeString(proposerHexAddress)
	if err != nil {
		return err
	}
	consAddr := sdk.ConsAddress(proposerBz)
	val, err := sk.ValidatorByConsAddr(ctx, consAddr)

	if err != nil {
		return err
	}

	if val == nil {
		return fmt.Errorf("no validator found by consAddr %s", consAddr.String())
	}

	powFromTokens := sk.TokensToConsensusPower(ctx, val.GetTokens())
	if powFromTokens == 0 {
		return fmt.Errorf("computed power == 0; nothing to slash")
	}

	slashFraction, err := slk.SlashFractionDowntime(ctx)
	if err != nil {
		return err
	}

	return slk.Slash(ctx, consAddr, slashFraction, powFromTokens, height)
}
