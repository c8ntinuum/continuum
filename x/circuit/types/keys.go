package types

const (
	ModuleName = "circuit"

	StoreKey = ModuleName

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
