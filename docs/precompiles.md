# Continuum precompiles

Precompiles are **natively executed system interfaces** exposed at fixed addresses. They are used when:

- Computation is too costly or complex in EVM bytecode (e.g., SNARK verification, advanced signature schemes), or  
- Contracts must safely interface with **native chain modules** (bank, staking, IBC, etc.) without re-implementing them in Solidity.

Continuum's precompiles are organized into the following families:

- Cross-chain verification & interoperability primitives  
- Native module access from Solidity  
- Cryptographic constructions  
- Hashing surfaces  
- Data handling & ergonomics  

---

## Cross-chain verification & interoperability primitives

These precompiles enable **trust-minimized cross-chain communication** by exposing light-client semantics, standardized token transfer logic, and zkVM-based verification.

| Precompile           | Purpose                                   | Description |
| -------------------- | ------------------------------------------ | ----------- |
| `ics02`              | Light-client / client semantics           | Enables contracts to participate in light-client style verification flows: updating verified remote consensus views and validating proof-carrying statements about remote state. Foundational for trust-minimized message passing. |
| `ics20`              | IBC fungible token transfers              | Exposes standardized cross-chain token transfer logic to EVM contracts, supporting composable cross-chain payments and asset movement without application-specific bridging logic. |
| `solanatx`           | Solana transaction / instruction parsing  | Provides a canonical parser surface so EVM contracts can interpret Solana transactions and instructions as structured data. Reduces ad-hoc parsing risk and enables principled validation of Solana-side actions within cross-chain workflows. |
| `sp1verifiergroth16` | Groth16 proof verification (SP1 zkVM)     | Verifies Groth16 proofs produced by SP1-based programs, enabling "prove off-chain, verify on-chain" designs for expensive verification tasks (e.g., zk light-client updates or succinct state attestations). |
| `sp1verifierplonk`   | Plonk proof verification (SP1 zkVM)       | Verifies Plonk proofs produced by SP1-based programs. Supporting multiple proof systems gives protocol designers flexibility over proof size, verification cost, and tooling constraints. |

---

## Native module access from Solidity

These precompiles make **Cosmos-native modules** and chain economics directly accessible from Solidity, enabling hybrid Cosmos–EVM applications.

| Precompile     | Purpose                                | Description |
| -------------- | -------------------------------------- | ----------- |
| `bank`         | Native asset accounting & transfers    | Exposes canonical balance, supply, and denomination logic to contracts, allowing EVM applications to treat the chain's native tokens and accounting rules as first-class. |
| `erc20`        | ERC-20 surface for native assets       | Bridges Ethereum's de facto token interface into the native token model so existing DeFi contracts and tooling can interoperate with Cosmos-native assets. |
| `werc20`       | Wrapped-native token interface         | Provides a canonical "WETH-like" wrapper pattern for the native token, supporting ERC-20-only contract flows while preserving native-asset utility. |
| `staking`      | Delegation-state interface             | Makes staking operations and staking state queryable by contracts, enabling designs in liquid staking, vault automation, and protocol-owned staking. |
| `distribution` | Rewards accounting & withdrawal        | Supports programmatic reward handling (e.g., auto-compounding strategies) and makes reward flows composable at the smart-contract layer. |
| `slashing`     | Fault / penalty observability          | Exposes slashing and jailing semantics so contracts can incorporate validator risk into allocation, rebalancing, or safety constraints. |
| `gov`          | Governance interface                   | Enables contracts to participate in proposal / voting workflows or to build governance-driven automation where execution is coupled to on-chain decisions. |
| `valrewards`   | Custom incentive surface               | Provides a specialized interface for validator reward logic, useful for experimenting with incentive design and interoperability-aligned rewards. |
| `bech32`       | Address representation bridge          | Enables robust conversion and parsing between bech32 and EVM-style address representations—important for correctness in cross-domain UX and contract logic. |

---

## Cryptographic constructions

These precompiles provide **advanced signature schemes, threshold protocols, and verifiable randomness**, turning Continuum into a platform for cryptographic protocol research and deployment.

| Precompile   | Purpose                                  | Description |
| ------------ | ---------------------------------------- | ----------- |
| `ed25519`    | EdDSA (Ed25519) verification             | Enables verification of Ed25519 signatures, important for ecosystems and protocols that rely on EdDSA-based authentication (e.g., Solana-style keys). |
| `p256`       | NIST P-256 verification                  | Provides a high-assurance mainstream curve for signature verification (e.g., enterprise crypto stacks and modern authentication ecosystems). |
| `schnorr`    | Schnorr over secp256k1                   | Supports Schnorr verification in a widely deployed curve setting, enabling aggregation-friendly and advanced authorization constructions. |
| `schnorrkel` | Schnorrkel / sr25519-style verification  | Supports verification in ecosystems that use Schnorrkel-style signatures, expanding the set of externally verifiable assertions. |
| `frost`      | Threshold signature share verification   | Exposes FROST primitives for protocols that require threshold signing workflows and auditable share verification. |
| `ecvrf`      | Verifiable Random Functions (VRFs)       | Enables publicly verifiable randomness derived from secret keys, useful for unbiased selection, lotteries, leader election, and randomness-beacon-style designs. |
| `pqmldsa`    | Post-quantum (ML-DSA) verification       | Adds a PQ signature option aligned with NIST standardization for long-horizon security planning and experimentation. |
| `pqslhdsa`   | Post-quantum (SLH-DSA) verification      | Adds a hash-based PQ signature option as a complementary design point relative to lattice-based signatures. |

---

## Hashing surfaces (standard + ZK-native)

Hash precompiles provide **standardized and ZK-efficient hashing** aligned with modern proof systems and protocol engineering.

| Precompile     | Purpose                                  | Description |
| -------------- | ---------------------------------------- | ----------- |
| `sha3hash`     | SHA-3 family hashing                     | Standardized hashing for commitments and compatibility with widely used cryptographic constructions. |
| `blake2bhash`  | BLAKE2b hashing                          | High-performance hashing commonly used in modern protocols and cryptographic engineering. |
| `poseidonhash` | Poseidon (ZK-efficient) hashing          | Optimized for constraint efficiency in zero-knowledge circuits; useful when on-chain logic must match off-chain proving systems' native hash choices. |
| `gnarkhash`    | ZK-oriented hashing (gnark-crypto aligned) | Practical hash utilities commonly used in proof systems and zk engineering toolchains. |

---

## Data handling & ergonomics

These precompiles improve **developer ergonomics** and **UX** for multi-domain applications by standardizing data parsing and address handling.

| Precompile     | Purpose                         | Description |
| -------------- | ------------------------------- | ----------- |
| `json`         | Structured payload parsing      | Enables contracts to interpret structured messages without brittle custom parsers, supporting standardized metadata and proof-carrying message formats. |
| `addresstable` | Address indirection / compression | Supports short-hands for frequently used addresses, reducing calldata size and improving gas efficiency in repetitive interaction patterns. |

---

## Design notes

- Precompiles are implemented as **native code** for gas-efficiency and correctness.  
- Address assignments are **stable by design** to preserve composability across upgrades.  
- When building protocols that depend on specific precompile behavior (e.g., ZK verification, PQ signatures), refer to the versioned specification and tests to ensure reproducibility across network upgrades.
