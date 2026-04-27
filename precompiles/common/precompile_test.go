package common

import (
	"math/big"
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/x/vm/statedb"
	vmmocks "github.com/cosmos/evm/x/vm/types/mocks"

	"cosmossdk.io/log"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type nativeActionKeeper struct {
	*vmmocks.EVMKeeper
	storeKeys map[string]*storetypes.KVStoreKey
}

func newNativeActionKeeper(storeKey *storetypes.KVStoreKey) *nativeActionKeeper {
	return &nativeActionKeeper{
		EVMKeeper: vmmocks.NewEVMKeeper(),
		storeKeys: map[string]*storetypes.KVStoreKey{
			storeKey.Name(): storeKey,
		},
	}
}

func (k nativeActionKeeper) KVStoreKeys() map[string]*storetypes.KVStoreKey {
	return k.storeKeys
}

func newNativeActionStateDB(t *testing.T) *statedb.StateDB {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey("precompile-test")
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger()).
		WithEventManager(sdk.NewEventManager())

	return statedb.New(ctx, newNativeActionKeeper(storeKey), statedb.NewEmptyTxConfig())
}

func newNativeActionEVM(stateDB *statedb.StateDB) *vm.EVM {
	return vm.NewEVM(vm.BlockContext{
		BlockNumber: big.NewInt(1),
		BaseFee:     big.NewInt(0),
		BlobBaseFee: big.NewInt(0),
		Difficulty:  big.NewInt(0),
	}, stateDB, params.TestChainConfig, vm.Config{})
}

func TestRunNativeActionRecoversUnexpectedPanics(t *testing.T) {
	stateDB := newNativeActionStateDB(t)
	precompile := Precompile{
		KvGasConfig:          storetypes.KVGasConfig(),
		TransientKVGasConfig: storetypes.TransientGasConfig(),
		ContractAddress:      ethcommon.BigToAddress(big.NewInt(1)),
	}
	contract := vm.NewContract(
		ethcommon.BigToAddress(big.NewInt(2)),
		precompile.Address(),
		uint256.NewInt(0),
		100_000,
		nil,
	)

	_, err := precompile.runNativeAction(newNativeActionEVM(stateDB), contract, func(sdk.Context) ([]byte, error) {
		panic("boom")
	})

	require.ErrorIs(t, err, vm.ErrExecutionReverted)
}

func TestRunNativeActionPreservesOutOfGas(t *testing.T) {
	stateDB := newNativeActionStateDB(t)
	precompile := Precompile{
		KvGasConfig:          storetypes.KVGasConfig(),
		TransientKVGasConfig: storetypes.TransientGasConfig(),
		ContractAddress:      ethcommon.BigToAddress(big.NewInt(1)),
	}
	contract := vm.NewContract(
		ethcommon.BigToAddress(big.NewInt(2)),
		precompile.Address(),
		uint256.NewInt(0),
		1,
		nil,
	)

	_, err := precompile.runNativeAction(newNativeActionEVM(stateDB), contract, func(ctx sdk.Context) ([]byte, error) {
		ctx.GasMeter().ConsumeGas(2, "force out of gas")
		return nil, nil
	})

	require.ErrorIs(t, err, vm.ErrOutOfGas)
}
