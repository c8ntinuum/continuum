package types

const (
	ModuleName = "ibcratelimiterext"
	// StoreKey must not share a prefix with other KV stores. In particular,
	// it cannot start with "ibc" (IBC core) or "ratelimit" (ibc-apps rate limiter).
	StoreKey  = "rlext"
	RouterKey = ModuleName
)

const (
	prefixParams = iota + 1
)

var (
	KeyParams = []byte{prefixParams}
)
