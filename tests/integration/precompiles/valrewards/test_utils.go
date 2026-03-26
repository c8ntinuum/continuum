package valrewards

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	abcitypes "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/evm/testutil/integration/evm/factory"
	testutiltypes "github.com/cosmos/evm/testutil/types"
	vrtypes "github.com/cosmos/evm/x/valrewards/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	errorsmod "cosmossdk.io/errors"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

// callType constants to differentiate between direct calls and calls through a contract.
const (
	directCall = iota + 1
)

// ContractData is a helper struct to hold the addresses and ABIs for the
// different contract instances that are subject to testing here.
type ContractData struct {
	precompileAddr common.Address
	precompileABI  abi.ABI
}

// getTxAndCallArgs returns tx and call args for direct precompile calls.
func getTxAndCallArgs(
	callType int,
	contractData ContractData,
	methodName string,
	args ...interface{},
) (evmtypes.EvmTxArgs, testutiltypes.CallArgs) {
	txArgs := evmtypes.EvmTxArgs{}
	callArgs := testutiltypes.CallArgs{}

	switch callType {
	case directCall:
		txArgs.To = &contractData.precompileAddr
		callArgs.ContractABI = contractData.precompileABI
	default:
		panic("unsupported call type")
	}

	callArgs.MethodName = methodName
	callArgs.Args = args

	return txArgs, callArgs
}

func (is *IntegrationTestSuite) advanceBlocks(count int64) error {
	for i := int64(0); i < count; i++ {
		if err := is.network.NextBlock(); err != nil {
			return err
		}
	}
	return nil
}

func (is *IntegrationTestSuite) advanceToRewardsEpoch() error {
	// Ensure epoch 0 is settled and rewards are recorded.
	return is.advanceBlocks(vrtypes.BLOCKS_IN_EPOCH + 2)
}

func (is *IntegrationTestSuite) commitContractCall(
	priv cryptotypes.PrivKey,
	txArgs evmtypes.EvmTxArgs,
	callArgs testutiltypes.CallArgs,
) (abcitypes.ExecTxResult, *evmtypes.MsgEthereumTxResponse, error) {
	input, err := factory.GenerateContractCallArgs(callArgs)
	if err != nil {
		return abcitypes.ExecTxResult{}, nil, errorsmod.Wrap(err, "failed to generate contract call args")
	}
	txArgs.Input = input

	signedTx, err := is.factory.GenerateSignedEthTx(priv, txArgs)
	if err != nil {
		return abcitypes.ExecTxResult{}, nil, errorsmod.Wrap(err, "failed to generate signed ethereum tx")
	}

	txBytes, err := is.factory.EncodeTx(signedTx)
	if err != nil {
		return abcitypes.ExecTxResult{}, nil, errorsmod.Wrap(err, "failed to encode ethereum tx")
	}

	blockRes, err := is.network.NextBlockWithTxs(txBytes)
	if err != nil {
		return abcitypes.ExecTxResult{}, nil, errorsmod.Wrap(err, "failed to include tx in block")
	}
	if len(blockRes.TxResults) != 1 {
		return abcitypes.ExecTxResult{}, nil, fmt.Errorf("expected 1 tx result, got %d", len(blockRes.TxResults))
	}
	res := *blockRes.TxResults[0]
	if !res.IsOK() {
		return res, nil, fmt.Errorf("tx failed with Code: %d, Logs: %s", res.Code, res.Log)
	}

	ethRes, err := evmtypes.DecodeTxResponse(res.Data)
	if err != nil {
		return res, nil, errorsmod.Wrap(err, "failed to decode ethereum tx response")
	}

	return res, ethRes, nil
}

func bigInt(amount string) *big.Int {
	out, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		panic("invalid big int")
	}
	return out
}
