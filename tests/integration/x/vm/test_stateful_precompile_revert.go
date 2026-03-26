package vm

import (
	"errors"
	"math/big"

	cmn "github.com/cosmos/evm/precompiles/common"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core"
	gethvm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type rollbackMode int

const (
	rollbackModeSuccess rollbackMode = iota
	rollbackModeError
	rollbackModeOutOfGas
)

// rollbackRegressionPrecompile writes directly to the Cosmos KV store through the EVM keeper
// to mimic a stateful precompile mutating native state before succeeding or failing.
type rollbackRegressionPrecompile struct {
	cmn.Precompile

	mode       rollbackMode
	writeState func(ctx sdk.Context)
}

func (p rollbackRegressionPrecompile) RequiredGas(_ []byte) uint64 {
	return 0
}

func (p rollbackRegressionPrecompile) Run(evm *gethvm.EVM, contract *gethvm.Contract, readonly bool) ([]byte, error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		p.writeState(ctx)

		switch p.mode {
		case rollbackModeError:
			return nil, errors.New("forced precompile failure")
		case rollbackModeOutOfGas:
			ctx.GasMeter().ConsumeGas(contract.Gas+1, "forced out of gas")
		}

		return []byte{0x1}, nil
	})
}

func (s *KeeperTestSuite) TestStatefulPrecompileFailureRevertsMultiStoreWrites() {
	testCases := []struct {
		name        string
		mode        rollbackMode
		expectValue bool
	}{
		{
			name:        "successful precompile persists native write",
			mode:        rollbackModeSuccess,
			expectValue: true,
		},
		{
			name:        "precompile error reverts native write",
			mode:        rollbackModeError,
			expectValue: false,
		},
		{
			name:        "precompile out of gas reverts native write",
			mode:        rollbackModeOutOfGas,
			expectValue: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			db := s.Network.GetStateDB()
			evmKeeper := s.Network.App.GetEVMKeeper()

			precompileAddr := common.HexToAddress("0x0000000000000000000000000000000000000900")
			storageAddr := common.HexToAddress("0x0000000000000000000000000000000000000999")
			storageKey := common.HexToHash("0x1")
			storageValue := common.HexToHash("0xbeef")
			caller := s.Keyring.GetAddr(0)

			precompile := rollbackRegressionPrecompile{
				Precompile: cmn.Precompile{
					ContractAddress: precompileAddr,
				},
				mode: tc.mode,
				writeState: func(ctx sdk.Context) {
					evmKeeper.SetState(ctx, storageAddr, storageKey, storageValue.Bytes())
				},
			}

			evm := s.newRegressionEVM(db, map[common.Address]gethvm.PrecompiledContract{
				precompileAddr: precompile,
			})

			ret, _, err := evm.Call(caller, precompileAddr, nil, 100_000, uint256.NewInt(0))
			if tc.expectValue {
				s.Require().NoError(err)
				s.Require().Equal([]byte{0x1}, ret)
			} else {
				s.Require().Error(err)
			}

			cacheCtx, cacheErr := db.GetCacheContext()
			s.Require().NoError(cacheErr)

			expected := common.Hash{}
			if tc.expectValue {
				expected = storageValue
			}

			s.Require().Equal(expected, evmKeeper.GetState(cacheCtx, storageAddr, storageKey))

			s.Require().NoError(db.Commit())
			s.Require().Equal(expected, evmKeeper.GetState(s.Network.GetContext(), storageAddr, storageKey))
		})
	}
}

func (s *KeeperTestSuite) newRegressionEVM(
	stateDB gethvm.StateDB,
	precompiles gethvm.PrecompiledContracts,
) *gethvm.EVM {
	ctx := s.Network.GetContext()
	random := common.Hash{}

	evm := gethvm.NewEVM(
		gethvm.BlockContext{
			CanTransfer: gethcore.CanTransfer,
			Transfer:    gethcore.Transfer,
			GetHash: func(uint64) common.Hash {
				return common.Hash{}
			},
			Coinbase:    common.Address{},
			GasLimit:    1_000_000,
			BlockNumber: big.NewInt(ctx.BlockHeight()),
			Time:        uint64(ctx.BlockTime().Unix()), //nolint:gosec
			Difficulty:  big.NewInt(0),
			BaseFee:     big.NewInt(0),
			Random:      &random,
		},
		stateDB,
		evmtypes.GetEthChainConfig(),
		gethvm.Config{},
	)
	evm.SetPrecompiles(precompiles)

	return evm
}
