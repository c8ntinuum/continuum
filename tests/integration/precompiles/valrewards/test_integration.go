package valrewards

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	cmn "github.com/cosmos/evm/precompiles/common"
	vrprecompile "github.com/cosmos/evm/precompiles/valrewards"
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

		sender        keyring.Key
		contractData  ContractData
		epoch         uint64
		validatorAddr string
	)

	_ = Describe("ValRewards Precompile -", func() {
		BeforeEach(func() {
			is = NewIntegrationTestSuite(create, options...)
			is.SetupTest()

			sender = is.keyring.GetKey(0)
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

		It("deposits and claims rewards via precompile", func() {
			rewardsPerEpoch := vrtypes.GetRewardsPerEpoch()
			depositAmount := new(big.Int).Mul(rewardsPerEpoch.Amount.BigInt(), big.NewInt(2))

			depositCoin := cmn.Coin{
				Denom:  rewardsPerEpoch.Denom,
				Amount: depositAmount,
			}

			txArgs, callArgs := getTxAndCallArgs(directCall, contractData, vrprecompile.DepositValidatorRewardsPoolMethod, sender.Addr, depositCoin)
			_, _, err := is.commitContractCall(sender.Priv, txArgs, callArgs)
			Expect(err).ToNot(HaveOccurred(), "unexpected error depositing rewards pool")

			txArgs, callArgs = getTxAndCallArgs(directCall, contractData, vrprecompile.ClaimRewardsMethod, sender.Addr, uint32(1), epoch)
			_, _, err = is.commitContractCall(sender.Priv, txArgs, callArgs)
			Expect(err).ToNot(HaveOccurred(), "unexpected error claiming rewards")

			txArgs, callArgs = getTxAndCallArgs(directCall, contractData, vrprecompile.DelegationRewardsMethod, sender.Addr, epoch)
			ethRes, err := is.factory.QueryContract(txArgs, callArgs, 1_000_000)
			Expect(err).ToNot(HaveOccurred(), "unexpected error querying delegation rewards")

			var outRewards struct {
				Rewards cmn.Coin
			}
			err = vrprecompile.ABI.UnpackIntoInterface(&outRewards, vrprecompile.DelegationRewardsMethod, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack delegation rewards")
			Expect(outRewards.Rewards.Amount.Sign()).To(Equal(0), "expected rewards to be cleared after claim")
		})
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "ValRewards Precompile Suite")
}
