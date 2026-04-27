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

type mixedRollbackMode int

const (
	mixedRollbackModeSuccess mixedRollbackMode = iota
	mixedRollbackModeErrorAfterNative
	mixedRollbackModeErrorAfterEVM
	mixedRollbackModeOutOfGasAfterEVM
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

// mixedRollbackRegressionPrecompile performs one native bank write and one EVM storage write
// in the same precompile call so the rollback path can be checked across both state layers.
type mixedRollbackRegressionPrecompile struct {
	cmn.Precompile

	mode         mixedRollbackMode
	bankKeeper   cmn.BankKeeper
	sender       sdk.AccAddress
	receiver     sdk.AccAddress
	transfer     sdk.Coins
	storageAddr  common.Address
	storageKey   common.Hash
	storageValue common.Hash
}

func (p mixedRollbackRegressionPrecompile) RequiredGas(_ []byte) uint64 {
	return 0
}

func (p mixedRollbackRegressionPrecompile) Run(evm *gethvm.EVM, contract *gethvm.Contract, readonly bool) ([]byte, error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		if err := p.bankKeeper.SendCoins(ctx, p.sender, p.receiver, p.transfer); err != nil {
			return nil, err
		}

		if p.mode == mixedRollbackModeErrorAfterNative {
			return nil, errors.New("forced precompile failure after native write")
		}

		evm.StateDB.SetState(p.storageAddr, p.storageKey, p.storageValue)

		switch p.mode {
		case mixedRollbackModeErrorAfterEVM:
			return nil, errors.New("forced precompile failure after evm write")
		case mixedRollbackModeOutOfGasAfterEVM:
			ctx.GasMeter().ConsumeGas(contract.Gas+1, "forced out of gas after mixed writes")
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

func (s *KeeperTestSuite) TestStatefulPrecompileFailureRevertsMixedNativeAndEVMWrites() {
	testCases := []struct {
		name          string
		mode          mixedRollbackMode
		expectPersist bool
	}{
		{
			name:          "successful precompile persists native and evm writes",
			mode:          mixedRollbackModeSuccess,
			expectPersist: true,
		},
		{
			name:          "precompile error after native write reverts both state layers",
			mode:          mixedRollbackModeErrorAfterNative,
			expectPersist: false,
		},
		{
			name:          "precompile error after evm write reverts both state layers",
			mode:          mixedRollbackModeErrorAfterEVM,
			expectPersist: false,
		},
		{
			name:          "precompile out of gas after evm write reverts both state layers",
			mode:          mixedRollbackModeOutOfGasAfterEVM,
			expectPersist: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			precompileAddr := common.HexToAddress("0x0000000000000000000000000000000000000901")
			storageAddr := common.HexToAddress("0x0000000000000000000000000000000000000998")
			storageKey := common.HexToHash("0x2")
			storageValue := common.HexToHash("0xcafe")
			senderAddr := s.Keyring.GetAddr(0)
			senderAcc := s.Keyring.GetAccAddr(0)
			receiverAcc := s.Keyring.GetAccAddr(1)
			denom := s.Network.GetBaseDenom()
			transferAmount := sdk.NewCoins(sdk.NewInt64Coin(denom, 100))
			bankKeeper := s.Network.App.GetBankKeeper()
			evmKeeper := s.Network.App.GetEVMKeeper()

			// Ensure the sender has spendable funds regardless of the wider suite defaults.
			s.Require().NoError(s.Network.FundAccount(senderAcc, sdk.NewCoins(sdk.NewInt64Coin(denom, 1_000))))

			db := s.Network.GetStateDB()

			senderBefore := bankKeeper.GetBalance(s.Network.GetContext(), senderAcc, denom).Amount
			receiverBefore := bankKeeper.GetBalance(s.Network.GetContext(), receiverAcc, denom).Amount

			precompile := mixedRollbackRegressionPrecompile{
				Precompile: cmn.Precompile{
					ContractAddress:       precompileAddr,
					BalanceHandlerFactory: cmn.NewBalanceHandlerFactory(bankKeeper),
				},
				mode:         tc.mode,
				bankKeeper:   bankKeeper,
				sender:       senderAcc,
				receiver:     receiverAcc,
				transfer:     transferAmount,
				storageAddr:  storageAddr,
				storageKey:   storageKey,
				storageValue: storageValue,
			}

			evm := s.newRegressionEVM(db, map[common.Address]gethvm.PrecompiledContract{
				precompileAddr: precompile,
			})

			ret, _, err := evm.Call(senderAddr, precompileAddr, nil, 100_000, uint256.NewInt(0))
			if tc.expectPersist {
				s.Require().NoError(err)
				s.Require().Equal([]byte{0x1}, ret)
			} else {
				s.Require().Error(err)
			}

			cacheCtx, cacheErr := db.GetCacheContext()
			s.Require().NoError(cacheErr)

			expectedStorage := common.Hash{}
			expectedSenderBalance := senderBefore
			expectedReceiverBalance := receiverBefore
			if tc.expectPersist {
				expectedStorage = storageValue
				expectedSenderBalance = senderBefore.Sub(transferAmount[0].Amount)
				expectedReceiverBalance = receiverBefore.Add(transferAmount[0].Amount)
			}

			s.Require().Equal(expectedStorage, db.GetState(storageAddr, storageKey))
			s.Require().True(expectedSenderBalance.Equal(bankKeeper.GetBalance(cacheCtx, senderAcc, denom).Amount))
			s.Require().True(expectedReceiverBalance.Equal(bankKeeper.GetBalance(cacheCtx, receiverAcc, denom).Amount))

			s.Require().NoError(db.Commit())
			s.Require().Equal(expectedStorage, evmKeeper.GetState(s.Network.GetContext(), storageAddr, storageKey))
			s.Require().True(expectedSenderBalance.Equal(bankKeeper.GetBalance(s.Network.GetContext(), senderAcc, denom).Amount))
			s.Require().True(expectedReceiverBalance.Equal(bankKeeper.GetBalance(s.Network.GetContext(), receiverAcc, denom).Amount))
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
