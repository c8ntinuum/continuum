package types

const (
	P256PrecompileAddress   = "0x0000000000000000000000000000000000000100"
	Bech32PrecompileAddress = "0x0000000000000000000000000000000000000400"
)

const (
	StakingPrecompileAddress      = "0x0000000000000000000000000000000000000800"
	DistributionPrecompileAddress = "0x0000000000000000000000000000000000000801"
	ICS20PrecompileAddress        = "0x0000000000000000000000000000000000000802"
	VestingPrecompileAddress      = "0x0000000000000000000000000000000000000803"
	BankPrecompileAddress         = "0x0000000000000000000000000000000000000804"
	GovPrecompileAddress          = "0x0000000000000000000000000000000000000805"
	SlashingPrecompileAddress     = "0x0000000000000000000000000000000000000806"
	ICS02PrecompileAddress        = "0x0000000000000000000000000000000000000807"
)

// c8ntinuum
const (
	Ed25519PrecompileAddress            = "0x0000000000000000000000000000000000000500"
	SP1VerifierGroth16PrecompileAddress = "0x0000000000000000000000000000000000000600"
	SP1VerifierPlonkPrecompileAddress   = "0x0000000000000000000000000000000000000700"
	JsonPrecompileAddress               = "0x0000000000000000000000000000000000000701"
	SolanaTxPrecompileAddress           = "0x0000000000000000000000000000000000000702"
	SchnorrPrecompileAddress            = "0x0000000000000000000000000000000000000703"
	SchnorrkelPrecompileAddress         = "0x0000000000000000000000000000000000000704"
	GnarkHashPrecompileAddress          = "0x0000000000000000000000000000000000000705"
	Sha3HashPrecompileAddress           = "0x0000000000000000000000000000000000000706"
	Blake2bPrecompileAddress            = "0x0000000000000000000000000000000000000707"
	EcvrfPrecompileAddress              = "0x0000000000000000000000000000000000000708"
	FrostPrecompileAddress              = "0x0000000000000000000000000000000000000709"
	AddressTablePrecompileAddress       = "0x0000000000000000000000000000000000000710"
	PoseidonHashPrecompileAddress       = "0x0000000000000000000000000000000000000711"
	PQMLDSAPrecompileAddress            = "0x0000000000000000000000000000000000000712"
	PQSLHDSAPrecompileAddress           = "0x0000000000000000000000000000000000000713"
	ValRewardsPrecompileAddress         = "0x0000000000000000000000000000000000000714"
	ReservedSlot15PrecompileAddress     = "0x0000000000000000000000000000000000000715"
	ReservedSlot16PrecompileAddress     = "0x0000000000000000000000000000000000000716"
	ReservedSlot17PrecompileAddress     = "0x0000000000000000000000000000000000000717"
	ReservedSlot18PrecompileAddress     = "0x0000000000000000000000000000000000000718"
	ReservedSlot19PrecompileAddress     = "0x0000000000000000000000000000000000000719"
	ReservedSlot20PrecompileAddress     = "0x0000000000000000000000000000000000000720"
	ReservedSlot21PrecompileAddress     = "0x0000000000000000000000000000000000000721"
	ReservedSlot22PrecompileAddress     = "0x0000000000000000000000000000000000000722"
	ReservedSlot23PrecompileAddress     = "0x0000000000000000000000000000000000000723"
	ReservedSlot24PrecompileAddress     = "0x0000000000000000000000000000000000000724"
	ReservedSlot25PrecompileAddress     = "0x0000000000000000000000000000000000000725"
	ReservedSlot26PrecompileAddress     = "0x0000000000000000000000000000000000000726"
	ReservedSlot27PrecompileAddress     = "0x0000000000000000000000000000000000000727"
	ReservedSlot28PrecompileAddress     = "0x0000000000000000000000000000000000000728"
	ReservedSlot29PrecompileAddress     = "0x0000000000000000000000000000000000000729"
	ReservedSlot30PrecompileAddress     = "0x0000000000000000000000000000000000000730"
	ReservedSlot31PrecompileAddress     = "0x0000000000000000000000000000000000000731"
	ReservedSlot32PrecompileAddress     = "0x0000000000000000000000000000000000000732"
	ReservedSlot33PrecompileAddress     = "0x0000000000000000000000000000000000000733"
	ReservedSlot34PrecompileAddress     = "0x0000000000000000000000000000000000000734"
	ReservedSlot35PrecompileAddress     = "0x0000000000000000000000000000000000000735"
	ReservedSlot36PrecompileAddress     = "0x0000000000000000000000000000000000000736"
	ReservedSlot37PrecompileAddress     = "0x0000000000000000000000000000000000000737"
	ReservedSlot38PrecompileAddress     = "0x0000000000000000000000000000000000000738"
	ReservedSlot39PrecompileAddress     = "0x0000000000000000000000000000000000000739"
	ReservedSlot40PrecompileAddress     = "0x0000000000000000000000000000000000000740"
	ReservedSlot41PrecompileAddress     = "0x0000000000000000000000000000000000000741"
	ReservedSlot42PrecompileAddress     = "0x0000000000000000000000000000000000000742"
	ReservedSlot43PrecompileAddress     = "0x0000000000000000000000000000000000000743"
	ReservedSlot44PrecompileAddress     = "0x0000000000000000000000000000000000000744"
	ReservedSlot45PrecompileAddress     = "0x0000000000000000000000000000000000000745"
	ReservedSlot46PrecompileAddress     = "0x0000000000000000000000000000000000000746"
	ReservedSlot47PrecompileAddress     = "0x0000000000000000000000000000000000000747"
	ReservedSlot48PrecompileAddress     = "0x0000000000000000000000000000000000000748"
	ReservedSlot49PrecompileAddress     = "0x0000000000000000000000000000000000000749"
	ReservedSlot50PrecompileAddress     = "0x0000000000000000000000000000000000000750"
)

// AvailableStaticPrecompiles defines the full list of all available EVM extension addresses.
//
// NOTE: To be explicit, this list does not include the dynamically registered EVM extensions
// like the ERC-20 extensions.
var AvailableStaticPrecompiles = []string{
	P256PrecompileAddress,
	Bech32PrecompileAddress,
	StakingPrecompileAddress,
	DistributionPrecompileAddress,
	ICS20PrecompileAddress,
	VestingPrecompileAddress,
	BankPrecompileAddress,
	GovPrecompileAddress,
	SlashingPrecompileAddress,
	ICS02PrecompileAddress,
	// c8ntinuum
	Ed25519PrecompileAddress,
	SP1VerifierGroth16PrecompileAddress,
	SP1VerifierPlonkPrecompileAddress,
	JsonPrecompileAddress,
	SolanaTxPrecompileAddress,
	SchnorrPrecompileAddress,
	SchnorrkelPrecompileAddress,
	GnarkHashPrecompileAddress,
	Sha3HashPrecompileAddress,
	Blake2bPrecompileAddress,
	EcvrfPrecompileAddress,
	FrostPrecompileAddress,
	AddressTablePrecompileAddress,
	PoseidonHashPrecompileAddress,
	PQMLDSAPrecompileAddress,
	PQSLHDSAPrecompileAddress,
	ValRewardsPrecompileAddress,
	ReservedSlot15PrecompileAddress,
	ReservedSlot16PrecompileAddress,
	ReservedSlot17PrecompileAddress,
	ReservedSlot18PrecompileAddress,
	ReservedSlot19PrecompileAddress,
	ReservedSlot20PrecompileAddress,
	ReservedSlot21PrecompileAddress,
	ReservedSlot22PrecompileAddress,
	ReservedSlot23PrecompileAddress,
	ReservedSlot24PrecompileAddress,
	ReservedSlot25PrecompileAddress,
	ReservedSlot26PrecompileAddress,
	ReservedSlot27PrecompileAddress,
	ReservedSlot28PrecompileAddress,
	ReservedSlot29PrecompileAddress,
	ReservedSlot30PrecompileAddress,
	ReservedSlot31PrecompileAddress,
	ReservedSlot32PrecompileAddress,
	ReservedSlot33PrecompileAddress,
	ReservedSlot34PrecompileAddress,
	ReservedSlot35PrecompileAddress,
	ReservedSlot36PrecompileAddress,
	ReservedSlot37PrecompileAddress,
	ReservedSlot38PrecompileAddress,
	ReservedSlot39PrecompileAddress,
	ReservedSlot40PrecompileAddress,
	ReservedSlot41PrecompileAddress,
	ReservedSlot42PrecompileAddress,
	ReservedSlot43PrecompileAddress,
	ReservedSlot44PrecompileAddress,
	ReservedSlot45PrecompileAddress,
	ReservedSlot46PrecompileAddress,
	ReservedSlot47PrecompileAddress,
	ReservedSlot48PrecompileAddress,
	ReservedSlot49PrecompileAddress,
	ReservedSlot50PrecompileAddress,
	// END c8ntinuum
}
