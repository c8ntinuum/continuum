package types

const (
	ModuleName = "ibcbreaker"

	StoreKey = "breaker"

	RouterKey = ModuleName
)

const (
	prefixParams = iota + 1
	prefixState
)

var (
	KeyParams = []byte{prefixParams}
	KeyState  = []byte{prefixState}
)
