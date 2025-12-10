package types

import (
	"fmt"
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	ibcutils "github.com/cosmos/evm/ibc"
	bankprecompile "github.com/cosmos/evm/precompiles/bank"
	"github.com/cosmos/evm/precompiles/bech32"
	cmn "github.com/cosmos/evm/precompiles/common"
	distprecompile "github.com/cosmos/evm/precompiles/distribution"
	govprecompile "github.com/cosmos/evm/precompiles/gov"
	ics02precompile "github.com/cosmos/evm/precompiles/ics02"
	ics20precompile "github.com/cosmos/evm/precompiles/ics20"
	"github.com/cosmos/evm/precompiles/p256"
	"github.com/cosmos/evm/precompiles/pqslhdsa"
	slashingprecompile "github.com/cosmos/evm/precompiles/slashing"
	stakingprecompile "github.com/cosmos/evm/precompiles/staking"
	erc20Keeper "github.com/cosmos/evm/x/erc20/keeper"
	transferkeeper "github.com/cosmos/evm/x/ibc/transfer/keeper"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	//c8ntinuum
	"github.com/cosmos/evm/precompiles/addresstable"
	"github.com/cosmos/evm/precompiles/blake2bhash"
	"github.com/cosmos/evm/precompiles/ecvrf"
	"github.com/cosmos/evm/precompiles/ed25519"
	"github.com/cosmos/evm/precompiles/frost"
	"github.com/cosmos/evm/precompiles/gnarkhash"
	json "github.com/cosmos/evm/precompiles/json"
	"github.com/cosmos/evm/precompiles/poseidonhash"
	"github.com/cosmos/evm/precompiles/pqmldsa"
	"github.com/cosmos/evm/precompiles/reserved"
	"github.com/cosmos/evm/precompiles/schnorr"
	"github.com/cosmos/evm/precompiles/schnorrkel"
	"github.com/cosmos/evm/precompiles/sha3hash"
	"github.com/cosmos/evm/precompiles/solanatx"
	"github.com/cosmos/evm/precompiles/sp1verifiergroth16"
	"github.com/cosmos/evm/precompiles/sp1verifierplonk"
	"github.com/cosmos/evm/precompiles/valrewards"
	vrkeeper "github.com/cosmos/evm/x/valrewards/keeper"
	// END c8ntinuum
)

type StaticPrecompiles map[common.Address]vm.PrecompiledContract

func NewStaticPrecompiles() StaticPrecompiles {
	return make(StaticPrecompiles)
}

func (s StaticPrecompiles) WithPraguePrecompiles() StaticPrecompiles {
	maps.Copy(s, vm.PrecompiledContractsPrague)
	return s
}

func (s StaticPrecompiles) WithP256Precompile() StaticPrecompiles {
	p256Precompile := &p256.Precompile{}
	s[p256Precompile.Address()] = p256Precompile
	return s
}

func (s StaticPrecompiles) WithBech32Precompile() StaticPrecompiles {
	bech32Precompile, err := bech32.NewPrecompile(bech32PrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate bech32 precompile: %w", err))
	}
	s[bech32Precompile.Address()] = bech32Precompile
	return s
}

func (s StaticPrecompiles) WithStakingPrecompile(
	stakingKeeper stakingkeeper.Keeper,
	bankKeeper cmn.BankKeeper,
	opts ...Option,
) StaticPrecompiles {
	options := defaultOptionals()
	for _, opt := range opts {
		opt(&options)
	}

	stakingPrecompile := stakingprecompile.NewPrecompile(
		stakingKeeper,
		stakingkeeper.NewMsgServerImpl(&stakingKeeper),
		stakingkeeper.NewQuerier(&stakingKeeper),
		bankKeeper,
		options.AddressCodec,
	)

	s[stakingPrecompile.Address()] = stakingPrecompile
	return s
}

func (s StaticPrecompiles) WithDistributionPrecompile(
	distributionKeeper distributionkeeper.Keeper,
	stakingKeeper stakingkeeper.Keeper,
	bankKeeper cmn.BankKeeper,
	opts ...Option,
) StaticPrecompiles {
	options := defaultOptionals()
	for _, opt := range opts {
		opt(&options)
	}

	distributionPrecompile := distprecompile.NewPrecompile(
		distributionKeeper,
		distributionkeeper.NewMsgServerImpl(distributionKeeper),
		distributionkeeper.NewQuerier(distributionKeeper),
		stakingKeeper,
		bankKeeper,
		options.AddressCodec,
	)

	s[distributionPrecompile.Address()] = distributionPrecompile
	return s
}

func (s StaticPrecompiles) WithICS02Precompile(
	codec codec.Codec,
	clientKeeper ibcutils.ClientKeeper,
) StaticPrecompiles {
	ibcClientPrecompile := ics02precompile.NewPrecompile(
		codec,
		clientKeeper,
	)

	s[ibcClientPrecompile.Address()] = ibcClientPrecompile
	return s
}

func (s StaticPrecompiles) WithICS20Precompile(
	bankKeeper cmn.BankKeeper,
	stakingKeeper stakingkeeper.Keeper,
	transferKeeper *transferkeeper.Keeper,
	channelKeeper *channelkeeper.Keeper,
) StaticPrecompiles {
	ibcTransferPrecompile := ics20precompile.NewPrecompile(
		bankKeeper,
		stakingKeeper,
		transferKeeper,
		channelKeeper,
	)

	s[ibcTransferPrecompile.Address()] = ibcTransferPrecompile
	return s
}

func (s StaticPrecompiles) WithBankPrecompile(
	bankKeeper cmn.BankKeeper,
	erc20Keeper *erc20Keeper.Keeper,
) StaticPrecompiles {
	bankPrecompile := bankprecompile.NewPrecompile(bankKeeper, erc20Keeper)
	s[bankPrecompile.Address()] = bankPrecompile
	return s
}

func (s StaticPrecompiles) WithGovPrecompile(
	govKeeper govkeeper.Keeper,
	bankKeeper cmn.BankKeeper,
	codec codec.Codec,
	opts ...Option,
) StaticPrecompiles {
	options := defaultOptionals()
	for _, opt := range opts {
		opt(&options)
	}

	govPrecompile := govprecompile.NewPrecompile(
		govkeeper.NewMsgServerImpl(&govKeeper),
		govkeeper.NewQueryServer(&govKeeper),
		bankKeeper,
		codec,
		options.AddressCodec,
	)

	s[govPrecompile.Address()] = govPrecompile
	return s
}

func (s StaticPrecompiles) WithSlashingPrecompile(
	slashingKeeper slashingkeeper.Keeper,
	bankKeeper cmn.BankKeeper,
	opts ...Option,
) StaticPrecompiles {
	options := defaultOptionals()
	for _, opt := range opts {
		opt(&options)
	}

	slashingPrecompile := slashingprecompile.NewPrecompile(
		slashingKeeper,
		slashingkeeper.NewMsgServerImpl(slashingKeeper),
		bankKeeper,
		options.ValidatorAddrCodec,
		options.ConsensusAddrCodec,
	)

	s[slashingPrecompile.Address()] = slashingPrecompile
	return s
}

// c8ntinuum
func (s StaticPrecompiles) WithEd25519Precompile() StaticPrecompiles {
	ed25519Precompile, err := ed25519.NewPrecompile(ed25519PrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate ed25519 precompile: %w", err))
	}
	s[ed25519Precompile.Address()] = ed25519Precompile
	return s
}

func (s StaticPrecompiles) WithGroth16Precompile() StaticPrecompiles {
	sp1verifierGroth16Precompile, err := sp1verifiergroth16.NewPrecompile(sp1verifierGroth16PrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate sp1verifier precompile: %w", err))
	}
	s[sp1verifierGroth16Precompile.Address()] = sp1verifierGroth16Precompile
	return s
}

func (s StaticPrecompiles) WithPlonkPrecompile() StaticPrecompiles {
	sp1verifierPlonkPrecompile, err := sp1verifierplonk.NewPrecompile(sp1verifierPlonkPrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate sp1verifier precompile: %w", err))
	}
	s[sp1verifierPlonkPrecompile.Address()] = sp1verifierPlonkPrecompile
	return s
}

func (s StaticPrecompiles) WithJsonPrecompile() StaticPrecompiles {
	jsonPrecompile, err := json.NewPrecompile(jsonPrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate json precompile: %w", err))
	}
	s[jsonPrecompile.Address()] = jsonPrecompile
	return s
}

func (s StaticPrecompiles) WithSolanaTxPrecompile(bankKeeper cmn.BankKeeper) StaticPrecompiles {
	solanatxPrecompile, err := solanatx.NewPrecompile(bankKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate solanatx precompile: %w", err))
	}
	s[solanatxPrecompile.Address()] = solanatxPrecompile
	return s
}

func (s StaticPrecompiles) WithSchnorrPrecompile() StaticPrecompiles {
	schnorrPrecompile, err := schnorr.NewPrecompile(schnorrPrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate schnorr precompile: %w", err))
	}
	s[schnorrPrecompile.Address()] = schnorrPrecompile
	return s
}

func (s StaticPrecompiles) WithSchnorrkelPrecompile() StaticPrecompiles {
	schnorrkelPrecompile, err := schnorrkel.NewPrecompile(schnorrkelPrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate schnorrkel precompile: %w", err))
	}
	s[schnorrkelPrecompile.Address()] = schnorrkelPrecompile
	return s
}

func (s StaticPrecompiles) WithGnarkPrecompile() StaticPrecompiles {
	gnarkHashPrecompile, err := gnarkhash.NewPrecompile(gnarkhashPrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate gnarkhash precompile: %w", err))
	}
	s[gnarkHashPrecompile.Address()] = gnarkHashPrecompile
	return s
}

func (s StaticPrecompiles) WithSha3Precompile() StaticPrecompiles {
	sha3hashPrecompile, err := sha3hash.NewPrecompile(sha3hashPrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate sha3hash precompile: %w", err))
	}
	s[sha3hashPrecompile.Address()] = sha3hashPrecompile
	return s
}

func (s StaticPrecompiles) WithBlake2bPrecompile() StaticPrecompiles {
	blake2bhashPrecompile, err := blake2bhash.NewPrecompile(blake2bhashPrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate blake2bhash precompile: %w", err))
	}
	s[blake2bhashPrecompile.Address()] = blake2bhashPrecompile
	return s
}

func (s StaticPrecompiles) WithEcvrfPrecompile() StaticPrecompiles {
	ecvrfPrecompile, err := ecvrf.NewPrecompile(ecvrfPrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate ecvrf precompile: %w", err))
	}
	s[ecvrfPrecompile.Address()] = ecvrfPrecompile
	return s
}

func (s StaticPrecompiles) WithFrostPrecompile() StaticPrecompiles {
	frostPrecompile, err := frost.NewPrecompile(frostPrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate frost precompile: %w", err))
	}
	s[frostPrecompile.Address()] = frostPrecompile
	return s
}

func (s StaticPrecompiles) WithAddressTablePrecompile(bankKeeper cmn.BankKeeper) StaticPrecompiles {
	addressTablePrecompile, err := addresstable.NewPrecompile(bankKeeper)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate addressTable precompile: %w", err))
	}
	s[addressTablePrecompile.Address()] = addressTablePrecompile
	return s
}

func (s StaticPrecompiles) WithPoseidonHashPrecompile() StaticPrecompiles {
	poseidonHashPrecompile, err := poseidonhash.NewPrecompile(poseidonHashBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate poseidonHash precompile: %w", err))
	}
	s[poseidonHashPrecompile.Address()] = poseidonHashPrecompile
	return s
}

func (s StaticPrecompiles) WithPQMLDSAPrecompile() StaticPrecompiles {
	pqmldsaPrecompile, err := pqmldsa.NewPrecompile(pqmldsaBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate pqmldsa precompile: %w", err))
	}
	s[pqmldsaPrecompile.Address()] = pqmldsaPrecompile
	return s
}

func (s StaticPrecompiles) WithPQSLHDSAPrecompile() StaticPrecompiles {
	pqslhdsaPrecompile, err := pqslhdsa.NewPrecompile(pqslhdsaBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate pqmldsa precompile: %w", err))
	}
	s[pqslhdsaPrecompile.Address()] = pqslhdsaPrecompile
	return s
}

func (s StaticPrecompiles) WithValidatorRewardsPrecompile(valrewardsKeeper vrkeeper.Keeper, accountkeeper accountkeeper.AccountKeeper, stakingKeeper cmn.StakingKeeper, bankKeeper cmn.BankKeeper) StaticPrecompiles {
	valrewardsPrecompile := valrewards.NewPrecompile(accountkeeper, stakingKeeper, bankKeeper, valrewardsKeeper)
	s[valrewardsPrecompile.Address()] = valrewardsPrecompile
	return s
}

func (s StaticPrecompiles) WithReservedPrecompiles() StaticPrecompiles {
	reserved15Precompile, _ := reserved.NewPrecompile(15)
	reserved16Precompile, _ := reserved.NewPrecompile(16)
	reserved17Precompile, _ := reserved.NewPrecompile(17)
	reserved18Precompile, _ := reserved.NewPrecompile(18)
	reserved19Precompile, _ := reserved.NewPrecompile(19)
	reserved20Precompile, _ := reserved.NewPrecompile(20)
	reserved21Precompile, _ := reserved.NewPrecompile(21)
	reserved22Precompile, _ := reserved.NewPrecompile(22)
	reserved23Precompile, _ := reserved.NewPrecompile(23)
	reserved24Precompile, _ := reserved.NewPrecompile(24)
	reserved25Precompile, _ := reserved.NewPrecompile(25)
	reserved26Precompile, _ := reserved.NewPrecompile(26)
	reserved27Precompile, _ := reserved.NewPrecompile(27)
	reserved28Precompile, _ := reserved.NewPrecompile(28)
	reserved29Precompile, _ := reserved.NewPrecompile(29)
	reserved30Precompile, _ := reserved.NewPrecompile(30)
	reserved31Precompile, _ := reserved.NewPrecompile(31)
	reserved32Precompile, _ := reserved.NewPrecompile(32)
	reserved33Precompile, _ := reserved.NewPrecompile(33)
	reserved34Precompile, _ := reserved.NewPrecompile(34)
	reserved35Precompile, _ := reserved.NewPrecompile(35)
	reserved36Precompile, _ := reserved.NewPrecompile(36)
	reserved37Precompile, _ := reserved.NewPrecompile(37)
	reserved38Precompile, _ := reserved.NewPrecompile(38)
	reserved39Precompile, _ := reserved.NewPrecompile(39)
	reserved40Precompile, _ := reserved.NewPrecompile(40)
	reserved41Precompile, _ := reserved.NewPrecompile(41)
	reserved42Precompile, _ := reserved.NewPrecompile(42)
	reserved43Precompile, _ := reserved.NewPrecompile(43)
	reserved44Precompile, _ := reserved.NewPrecompile(44)
	reserved45Precompile, _ := reserved.NewPrecompile(45)
	reserved46Precompile, _ := reserved.NewPrecompile(46)
	reserved47Precompile, _ := reserved.NewPrecompile(47)
	reserved48Precompile, _ := reserved.NewPrecompile(48)
	reserved49Precompile, _ := reserved.NewPrecompile(49)
	reserved50Precompile, _ := reserved.NewPrecompile(50)
	s[reserved15Precompile.Address()] = reserved15Precompile
	s[reserved16Precompile.Address()] = reserved16Precompile
	s[reserved17Precompile.Address()] = reserved17Precompile
	s[reserved18Precompile.Address()] = reserved18Precompile
	s[reserved19Precompile.Address()] = reserved19Precompile
	s[reserved20Precompile.Address()] = reserved20Precompile
	s[reserved21Precompile.Address()] = reserved21Precompile
	s[reserved22Precompile.Address()] = reserved22Precompile
	s[reserved23Precompile.Address()] = reserved23Precompile
	s[reserved24Precompile.Address()] = reserved24Precompile
	s[reserved25Precompile.Address()] = reserved25Precompile
	s[reserved26Precompile.Address()] = reserved26Precompile
	s[reserved27Precompile.Address()] = reserved27Precompile
	s[reserved28Precompile.Address()] = reserved28Precompile
	s[reserved29Precompile.Address()] = reserved29Precompile
	s[reserved30Precompile.Address()] = reserved30Precompile
	s[reserved31Precompile.Address()] = reserved31Precompile
	s[reserved32Precompile.Address()] = reserved32Precompile
	s[reserved33Precompile.Address()] = reserved33Precompile
	s[reserved34Precompile.Address()] = reserved34Precompile
	s[reserved35Precompile.Address()] = reserved35Precompile
	s[reserved36Precompile.Address()] = reserved36Precompile
	s[reserved37Precompile.Address()] = reserved37Precompile
	s[reserved38Precompile.Address()] = reserved38Precompile
	s[reserved39Precompile.Address()] = reserved39Precompile
	s[reserved40Precompile.Address()] = reserved40Precompile
	s[reserved41Precompile.Address()] = reserved41Precompile
	s[reserved42Precompile.Address()] = reserved42Precompile
	s[reserved43Precompile.Address()] = reserved43Precompile
	s[reserved44Precompile.Address()] = reserved44Precompile
	s[reserved45Precompile.Address()] = reserved45Precompile
	s[reserved46Precompile.Address()] = reserved46Precompile
	s[reserved47Precompile.Address()] = reserved47Precompile
	s[reserved48Precompile.Address()] = reserved48Precompile
	s[reserved49Precompile.Address()] = reserved49Precompile
	s[reserved50Precompile.Address()] = reserved50Precompile
	return s
}

// END c8ntinuum
