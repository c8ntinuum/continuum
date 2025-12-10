# Pre-deployed contracts

Continuum ships with a curated set of **pre-deployed ecosystem contracts** at well-known addresses. This mirrors the practice of Ethereum L2s and modular stacks (e.g., OP Stack) and is intended to:

- Remove the "bootstrap tax" of deploying basic infrastructure.
- Ensure immediate compatibility with existing wallets, SDKs, and protocol tooling.

These contracts are **identical or bytecode-equivalent** to widely used deployments on Ethereum and major rollups, unless otherwise noted.

---

## Directory of pre-deployed contracts

| Name                        | Standard / Role                                      | Address                                      | Description |
| --------------------------- | ---------------------------------------------------- | -------------------------------------------- | ----------- |
| `Create2`                   | Deterministic deployment helper (EIP-1014)          | `0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2` | Provides predictable contract addresses for counterfactual and multi-chain deployments via CREATE2. |
| `Multicall3`                | Batched calls (read & write)                        | `0xcA11bde05977b3631167028862bE2a173976CA11` | Aggregates multiple calls into one transaction for efficient querying and execution. |
| `Permit2`                   | Shared token approvals (Uniswap)                    | `0x000000000022D473030F116dDEE9F6B43aC78BA3` | Centralized approval and permit flows for ERC-20 tokens, improving UX across many protocols. |
| `Safe singleton factory`    | Safe canonical factory                              | `0x914d7Fec6aaC8cd542e72Bca78B30650d45643d7` | Canonical factory for deploying Safe instances at predictable addresses; compatible with existing Safe tooling. |
| `EIP-2935 history storage`  | Historical block-hash storage                       | `0x0000F90827F1C53a10cb7A02335B175320002935` | Stores an extended window of block hashes in state, enabling protocols that require access to older blocks. |
| `EIP1820 registry`          | Interface registry / pseudo-introspection           | `0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24` | Standard on-chain registry mapping accounts to supported interfaces and implementers. |
| `EIP2470 singleton factory` | Deterministic singleton factory                     | `0xce0042B868300000d44A59004Da54A005ffdcf9f` | Factory that deploys bytecode to a deterministic address, enabling "global singleton" contracts across networks. |
| `CreateX`                   | Universal contract deployer                         | `0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed` | Universal deployer wrapping CREATE / CREATE2-style flows to simplify deterministic deployments. |
| `MultiSend`                 | Generic batch transaction executor (Safe)           | `0x998739BFdAAdde7C933B942a68053933098f9EDa` | Executes multiple operations atomically from Safe and smart-account contexts. |
| `MultiSendCallOnly`         | Batch executor without `DELEGATECALL`               | `0xA1dabEF33b3B82c7814B6D82A79e50F4AC44102B` | Safer batch executor that forbids `DELEGATECALL` by construction. |
| `UniversalSigValidator`     | EIP-6492 signature validator for pre-deploy wallets | `0x7dd271fa79df3a5feb99f73bebfa4395b2e4f4be` | Validates signatures for "counterfactual" smart accounts as standardized in EIP-6492. |
| `WETH9`                     | Canonical wrapped-native token                      | `0xc8ef4398664b2eed5ee560544f659083d98a3888` | ERC-20 wrapper for the native token, enabling direct use in existing DeFi protocols and tooling. |

---

## Design considerations

- **Bytecode parity** – Where possible, Continuum reuses **exact** on-chain bytecode from Ethereum mainnet deployments to maximize compatibility with existing tools and audits.
- **Stable addresses** – These addresses are intended to be **stable across network upgrades**. If a breaking change is required, it should be accompanied by clear migration tooling and documentation.
- **Safety** – Contracts like `MultiSendCallOnly` are chosen to provide safer defaults (e.g., no `DELEGATECALL`) while still supporting advanced account and batch-execution patterns.

---

## Usage notes

- You can safely **assume the presence** of these contracts when writing dApps and tooling for Continuum (e.g., defaulting to the known `Multicall3` address in SDKs).
- For testing and local development, the same addresses are instantiated by the node when you run `./local_node.sh`.
- If you rely on specific behavior (e.g., gas costs, revert semantics), refer to the upstream source repositories and the Continuum tests that pin expected behavior.
