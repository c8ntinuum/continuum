// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.17;

/// @dev The SolanaTx precompile contract's address.
address constant SOLANA_TX_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000702;

/// @dev The SolanaTx precompile contract's instance.
SolanaTxI constant SOLANA_TX_CONTRACT = SolanaTxI(SOLANA_TX_PRECOMPILE_ADDRESS);

/// @author ...
/// @title Solana Transaction Precompiled Contract
/// @dev Interface for interacting with the Solana transaction generator precompile.
interface SolanaTxI {
    /// @dev Account metadata required for Solana instructions.
    struct AccountMeta {
        bytes32 pubKey; // 32-byte public key
        bool isSigner; // Whether the account must sign the transaction
        bool isWritable; // Whether the account is allowed to be written to
    }

    /// @notice Emitted on each successful generation & persistence of a Solana tx.
    /// @param priorityLevel Priority level as a UTF-8 string.
    /// @param programKey Solana program public key (32 bytes). (indexed)
    /// @param signer Solana transaction signer public key (32 bytes). (indexed)
    /// @param methodParams Arbitrary method parameters including selector.
    /// @param accounts Array of Solana account metadata.
    /// @param latestBlockHash Latest Solana block hash as a UTF-8 string.
    /// @param txToSignBytes Unsigned Solana transaction bytes.
    /// @param txId 32-byte transaction identifier (hash of txToSignBytes). (indexed)
    event SolanaTxGenerated(
        string priorityLevel,
        bytes32 indexed programKey,
        bytes32 indexed signer,
        bytes methodParams,
        AccountMeta[] accounts,
        string latestBlockHash,
        bytes txToSignBytes,
        bytes32 indexed txId
    );

    /// @notice Generate a Solana transaction for signing.
    /// @param priorityLevel Priority level as a UTF-8 string.
    /// @param programKey Solana program public key (32 bytes).
    /// @param signer Solana transaction signer public key (32 bytes).
    /// @param methodParams Arbitrary method parameters including method selector as well.
    /// @param accounts Array of Solana account metadata.
    /// @param latestBlockHash Latest Solana block hash as a UTF-8 string.
    /// @return txToSignBytes Unsigned Solana transaction bytes.
    /// @return txId 32-byte transaction identifier (hash of txToSignBytes).
    function generateSolanaTx(
        string memory priorityLevel,
        bytes32 programKey,
        bytes32 signer,
        bytes memory methodParams,
        AccountMeta[] memory accounts,
        string memory latestBlockHash
    ) external returns (bytes memory txToSignBytes, bytes32 txId);

    /// @notice Retrieve a previously generated Solana transaction.
    /// @param txId 32-byte transaction identifier returned from generateSolanaTx.
    /// @return txToSignBytes Unsigned Solana transaction bytes.
    function getSolanaTx(
        bytes32 txId
    ) external view returns (bytes memory txToSignBytes);

    /// @notice Retrieve multiple Solana transactions with pagination.
    /// @param skip Number of entries to skip.
    /// @param limit Maximum number of entries to return.
    /// @return txs Array of unsigned Solana transaction bytes.
    function getSolanaTxs(
        uint256 skip,
        uint256 limit
    ) external view returns (bytes[] memory txs);
}
