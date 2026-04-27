package valrewards

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/suite"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/precompiles/testutil"
	vrprecompile "github.com/cosmos/evm/precompiles/valrewards"
	evmfactory "github.com/cosmos/evm/testutil/integration/evm/factory"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	"github.com/cosmos/evm/testutil/keyring"
	vrtypes "github.com/cosmos/evm/x/valrewards/types"
)

// IntegrationTestSuite is the implementation of the TestSuite interface for valrewards precompile
// integration tests.
type PrecompileTestSuite struct {
	suite.Suite

	create  network.CreateEvmApp
	options []network.ConfigOption
}

func NewPrecompileTestSuite(create network.CreateEvmApp, options ...network.ConfigOption) *PrecompileTestSuite {
	return &PrecompileTestSuite{
		create:  create,
		options: options,
	}
}

func (s *PrecompileTestSuite) SetupTest() {}

func TestPrecompileIntegrationTestSuite(t *testing.T, create network.CreateEvmApp, options ...network.ConfigOption) {
	var (
		is *IntegrationTestSuite

		sender          keyring.Key
		sponsoredCaller keyring.Key
		contractData    ContractData
		epoch           uint64
		validatorAddr   string
	)

	_ = Describe("ValRewards Precompile -", func() {
		BeforeEach(func() {
			is = NewIntegrationTestSuite(create, options...)
			is.SetupTest()

			sender = is.keyring.GetKey(0)
			sponsoredCaller = is.keyring.GetKey(1)
			epoch = 0
			validatorAddr = is.network.GetValidators()[0].OperatorAddress

			contractData = ContractData{
				precompileAddr: is.precompileAddr(),
				precompileABI:  vrprecompile.ABI,
			}

			err := is.advanceToRewardsEpoch()
			Expect(err).ToNot(HaveOccurred(), "failed to advance blocks")
		})

		It("returns rewards pool via precompile", func() {
			txArgs, callArgs := getTxAndCallArgs(directCall, contractData, vrprecompile.RewardsPoolMethod)
			ethRes, err := is.factory.QueryContract(txArgs, callArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying rewards pool")

			var out struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&out, vrprecompile.RewardsPoolMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack rewards pool")
		})

		It("returns outstanding rewards via precompile", func() {
			txArgs, callArgs := getTxAndCallArgs(directCall, contractData, vrprecompile.DelegationRewardsMethod, sender.Addr, epoch)
			ethRes, err := is.factory.QueryContract(txArgs, callArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying delegation rewards")

			var out struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&out, vrprecompile.DelegationRewardsMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack delegation rewards")
			Expect(out.Rewards.Amount).ToNot(Equal(big.NewInt(0)), "expected non-zero rewards")

			txArgs, callArgs = getTxAndCallArgs(directCall, contractData, vrprecompile.ValidatorOutstandingRewardsMethod, epoch, validatorAddr)
			ethRes, err = is.factory.QueryContract(txArgs, callArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying validator outstanding rewards")

			var outVal struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&outVal, vrprecompile.ValidatorOutstandingRewardsMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack validator outstanding rewards")
			Expect(outVal.Rewards.Amount).ToNot(Equal(big.NewInt(0)), "expected non-zero rewards")
		})

		It("deposits and lets a third party trigger a validator rewards claim via precompile", func() {
			rewardsPerEpoch := vrtypes.GetRewardsPerEpoch()
			depositAmount := new(big.Int).Mul(rewardsPerEpoch.Amount.BigInt(), big.NewInt(2))

			depositCoin := cmn.Coin{
				Denom:  rewardsPerEpoch.Denom,
				Amount: depositAmount,
			}

			txArgs, callArgs := getTxAndCallArgs(directCall, contractData, vrprecompile.DepositValidatorRewardsPoolMethod, sender.Addr, depositCoin)
			_, _, err := is.commitContractCall(sender.Priv, txArgs, callArgs)
			Expect(err).ToNot(HaveOccurred(), "unexpected error depositing rewards pool")

			txArgs, callArgs = getTxAndCallArgs(directCall, contractData, vrprecompile.DelegationRewardsMethod, sender.Addr, epoch)
			ethRes, err := is.factory.QueryContract(txArgs, callArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying delegation rewards before claim")

			var outstandingBefore struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&outstandingBefore, vrprecompile.DelegationRewardsMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack delegation rewards before claim")

			claimAmount := sdkmath.NewIntFromBigInt(outstandingBefore.Rewards.Amount)
			Expect(claimAmount.IsPositive()).To(BeTrue(), "expected non-zero claimable rewards before claim")

			validatorBalanceBefore, err := is.grpcHandler.GetBalanceFromBank(sender.AccAddr, rewardsPerEpoch.Denom)
			Expect(err).ToNot(HaveOccurred(), "unexpected error reading validator operator balance before claim")

			txArgs, callArgs = getTxAndCallArgs(directCall, contractData, vrprecompile.ClaimRewardsMethod, sender.Addr, epoch)
			_, _, err = is.commitContractCall(sponsoredCaller.Priv, txArgs, callArgs)
			Expect(err).ToNot(HaveOccurred(), "unexpected error allowing third-party caller to trigger claim")

			validatorBalanceAfter, err := is.grpcHandler.GetBalanceFromBank(sender.AccAddr, rewardsPerEpoch.Denom)
			Expect(err).ToNot(HaveOccurred(), "unexpected error reading validator operator balance after claim")
			Expect(validatorBalanceAfter.Balance.Amount).To(Equal(
				validatorBalanceBefore.Balance.Amount.Add(claimAmount),
			), "expected rewards to be paid to the target validator operator")

			txArgs, callArgs = getTxAndCallArgs(directCall, contractData, vrprecompile.DelegationRewardsMethod, sender.Addr, epoch)
			ethRes, err = is.factory.QueryContract(txArgs, callArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying delegation rewards")

			var outRewards struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&outRewards, vrprecompile.DelegationRewardsMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack delegation rewards")
			Expect(outRewards.Rewards.Amount.Sign()).To(Equal(0), "expected rewards to be cleared after claim")
		})

		It("keeps the rewards pool unchanged on low gas deposit", func() {
			rewardsPerEpoch := vrtypes.GetRewardsPerEpoch()
			depositAmount := new(big.Int).Mul(rewardsPerEpoch.Amount.BigInt(), big.NewInt(2))
			depositCoin := cmn.Coin{
				Denom:  rewardsPerEpoch.Denom,
				Amount: depositAmount,
			}

			balanceBefore, err := is.grpcHandler.GetBalanceFromBank(sender.AccAddr, rewardsPerEpoch.Denom)
			Expect(err).ToNot(HaveOccurred(), "unexpected error reading depositor balance before low-gas deposit")

			queryTxArgs, queryCallArgs := getTxAndCallArgs(directCall, contractData, vrprecompile.RewardsPoolMethod)
			ethRes, err := is.factory.QueryContract(queryTxArgs, queryCallArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying rewards pool before low-gas deposit")

			var poolBefore struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&poolBefore, vrprecompile.RewardsPoolMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack rewards pool before low-gas deposit")

			txArgs, callArgs := getTxAndCallArgs(directCall, contractData, vrprecompile.DepositValidatorRewardsPoolMethod, sender.Addr, depositCoin)
			input, err := evmfactory.GenerateContractCallArgs(callArgs)
			Expect(err).ToNot(HaveOccurred())
			txArgs.Input = input

			estimatedGas, err := is.factory.EstimateGasLimit(&sender.Addr, &txArgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(estimatedGas).To(BeNumerically(">", 1))
			txArgs.GasLimit = estimatedGas - 1

			outOfGasCheck := testutil.LogCheckArgs{ABIEvents: vrprecompile.ABI.Events}.WithErrContains(vm.ErrOutOfGas.Error())
			_, _, err = is.factory.CallContractAndCheckLogs(sender.Priv, txArgs, callArgs, outOfGasCheck)
			Expect(err).ToNot(HaveOccurred(), "unexpected error on low-gas deposit")

			balanceAfter, err := is.grpcHandler.GetBalanceFromBank(sender.AccAddr, rewardsPerEpoch.Denom)
			Expect(err).ToNot(HaveOccurred(), "unexpected error reading depositor balance after low-gas deposit")
			Expect(balanceAfter.Balance).To(Equal(balanceBefore.Balance))

			ethRes, err = is.factory.QueryContract(queryTxArgs, queryCallArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying rewards pool after low-gas deposit")

			var poolAfter struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&poolAfter, vrprecompile.RewardsPoolMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack rewards pool after low-gas deposit")
			Expect(poolAfter.Rewards).To(Equal(poolBefore.Rewards))
		})

		It("keeps outstanding rewards unchanged on low gas claim", func() {
			rewardsPerEpoch := vrtypes.GetRewardsPerEpoch()
			depositAmount := new(big.Int).Mul(rewardsPerEpoch.Amount.BigInt(), big.NewInt(2))
			depositCoin := cmn.Coin{
				Denom:  rewardsPerEpoch.Denom,
				Amount: depositAmount,
			}

			txArgs, callArgs := getTxAndCallArgs(directCall, contractData, vrprecompile.DepositValidatorRewardsPoolMethod, sender.Addr, depositCoin)
			_, _, err := is.commitContractCall(sender.Priv, txArgs, callArgs)
			Expect(err).ToNot(HaveOccurred(), "unexpected error depositing rewards pool before low-gas claim")

			queryTxArgs, queryCallArgs := getTxAndCallArgs(directCall, contractData, vrprecompile.DelegationRewardsMethod, sender.Addr, epoch)
			ethRes, err := is.factory.QueryContract(queryTxArgs, queryCallArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying delegation rewards before low-gas claim")

			var rewardsBefore struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&rewardsBefore, vrprecompile.DelegationRewardsMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack delegation rewards before low-gas claim")

			validatorBalanceBefore, err := is.grpcHandler.GetBalanceFromBank(sender.AccAddr, rewardsPerEpoch.Denom)
			Expect(err).ToNot(HaveOccurred(), "unexpected error reading validator balance before low-gas claim")

			txArgs, callArgs = getTxAndCallArgs(directCall, contractData, vrprecompile.ClaimRewardsMethod, sender.Addr, epoch)
			input, err := evmfactory.GenerateContractCallArgs(callArgs)
			Expect(err).ToNot(HaveOccurred())
			txArgs.Input = input

			estimatedGas, err := is.factory.EstimateGasLimit(&sponsoredCaller.Addr, &txArgs)
			Expect(err).ToNot(HaveOccurred())
			Expect(estimatedGas).To(BeNumerically(">", 1))
			txArgs.GasLimit = estimatedGas - 1

			outOfGasCheck := testutil.LogCheckArgs{ABIEvents: vrprecompile.ABI.Events}.WithErrContains(vm.ErrOutOfGas.Error())
			_, _, err = is.factory.CallContractAndCheckLogs(sponsoredCaller.Priv, txArgs, callArgs, outOfGasCheck)
			Expect(err).ToNot(HaveOccurred(), "unexpected error on low-gas claim")

			validatorBalanceAfter, err := is.grpcHandler.GetBalanceFromBank(sender.AccAddr, rewardsPerEpoch.Denom)
			Expect(err).ToNot(HaveOccurred(), "unexpected error reading validator balance after low-gas claim")
			Expect(validatorBalanceAfter.Balance).To(Equal(validatorBalanceBefore.Balance))

			ethRes, err = is.factory.QueryContract(queryTxArgs, queryCallArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying delegation rewards after low-gas claim")

			var rewardsAfter struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&rewardsAfter, vrprecompile.DelegationRewardsMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack delegation rewards after low-gas claim")
			Expect(rewardsAfter.Rewards).To(Equal(rewardsBefore.Rewards))
		})

		It("rejects claims when the target address is not a validator operator", func() {
			txArgs, callArgs := getTxAndCallArgs(directCall, contractData, vrprecompile.DelegationRewardsMethod, sender.Addr, epoch)
			ethRes, err := is.factory.QueryContract(txArgs, callArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying delegation rewards before invalid claim")

			var outstandingBefore struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&outstandingBefore, vrprecompile.DelegationRewardsMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack delegation rewards before invalid claim")
			Expect(outstandingBefore.Rewards.Amount.Sign()).ToNot(Equal(0), "expected outstanding rewards before invalid claim")

			txArgs, callArgs = getTxAndCallArgs(directCall, contractData, vrprecompile.ClaimRewardsMethod, sponsoredCaller.Addr, epoch)
			_, _, err = is.commitContractCall(sender.Priv, txArgs, callArgs)
			Expect(err).To(HaveOccurred(), "expected non-validator target claim to fail")
			Expect(err.Error()).To(ContainSubstring("target address must be a validator operator"))

			txArgs, callArgs = getTxAndCallArgs(directCall, contractData, vrprecompile.DelegationRewardsMethod, sender.Addr, epoch)
			ethRes, err = is.factory.QueryContract(txArgs, callArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying delegation rewards after invalid claim")

			var outstandingAfter struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&outstandingAfter, vrprecompile.DelegationRewardsMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack delegation rewards after invalid claim")
			Expect(outstandingAfter.Rewards.Amount).To(Equal(outstandingBefore.Rewards.Amount), "expected outstanding rewards to remain unchanged after invalid claim")
		})
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "ValRewards Precompile Suite")
}
