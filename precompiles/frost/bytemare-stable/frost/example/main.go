package main

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/evm/precompiles/frost/bytemare-stable/frost"
	"github.com/cosmos/evm/precompiles/frost/bytemare-stable/frost/debug"
	"github.com/cosmos/evm/precompiles/frost/bytemare-stable/secret-sharing/keys"
)

func main() {
	maxSigners := uint16(5)
	threshold := uint16(3)
	message := []byte("example message")
	ciphersuite := frost.Default

	// We assume you already have a pool of participants with distinct non-zero identifiers and their signing share.
	// The following block uses a centralised trusted dealer to do this, but it is strongly recommended to use
	// distributed key generation, e.g. from github.com/cosmos/evm/precompiles/frost/bytemare-stable/dkg, which is compatible with FROST.
	secretKeyShares, verificationKey, _ := debug.TrustedDealerKeygen(ciphersuite, nil, threshold, maxSigners)
	participantSecretKeyShares := secretKeyShares[:threshold]
	participants := make([]*frost.Signer, threshold)

	// At key generation, each participant must send their public key share to the coordinator, and the collection must
	// be broadcast to every participant.
	publicKeyShares := make([]*keys.PublicKeyShare, len(secretKeyShares))
	for i, sk := range secretKeyShares {
		publicKeyShares[i] = sk.Public()
		fmt.Println("publicKeyShares", i, hex.EncodeToString(publicKeyShares[i].Encode()))
	}

	// This is how to set up the Configuration for FROST, the same for every signer and the coordinator.
	configuration := &frost.Configuration{
		Ciphersuite:           ciphersuite,
		Threshold:             threshold,
		MaxSigners:            maxSigners,
		VerificationKey:       verificationKey,
		SignerPublicKeyShares: publicKeyShares,
	}

	if err := configuration.Init(); err != nil {
		panic(err)
	}

	// Create a participant on each instance
	for i, ks := range participantSecretKeyShares {
		signer, err := configuration.Signer(ks)
		if err != nil {
			panic(err)
		}

		participants[i] = signer
	}

	// Pre-commit
	commitments := make(frost.CommitmentList, threshold)
	for i, p := range participants {
		commitments[i] = p.Commit()
	}

	commitments.Sort()

	// Sign
	signatureShares := make([]*frost.SignatureShare, threshold)
	for i, p := range participants {
		var err error
		signatureShares[i], err = p.Sign(message, commitments)
		if err != nil {
			panic(err)
		}
	}

	// Everything above was a simulation of commitment and signing rounds to produce the signature shares.
	// The following shows how to aggregate these shares, and if verification fails, how to identify a misbehaving signer.

	// The coordinator assembles the shares. If the verify argument is set to true, AggregateSignatures will internally
	// verify each signature share and return an error on the first that is invalid. It will also verify whether the
	// output signature is valid.
	signature, err := configuration.AggregateSignatures(message, signatureShares, commitments, true)
	if err != nil {
		panic(err)
	}

	fmt.Println("VerifySignature.ciphersuite", ciphersuite)
	fmt.Println("VerifySignature.message", hex.EncodeToString(message))
	fmt.Println("VerifySignature.signature", hex.EncodeToString(signature.Encode()))
	fmt.Println("VerifySignature.verificationKey", hex.EncodeToString(verificationKey.Encode()))

	// Verify the signature and identify potential foul players. Note that since we set verify to true when calling
	// AggregateSignatures, the following is redundant.
	// Anyone can verify the signature given the ciphersuite parameter, message, and the group public key.
	if err = frost.VerifySignature(ciphersuite, message, signature, verificationKey); err != nil {
		fmt.Println(err)
		panic("Signature verification failed.")
	}

	// At this point one should try to identify which participant's signature share is invalid and act on it.
	// This verification is done as follows:
	for i, signatureShare := range signatureShares {

		fmt.Println(i, "VerifySignatureShare.ciphersuite", ciphersuite)
		fmt.Println(i, "VerifySignatureShare.threshold", threshold)
		fmt.Println(i, "VerifySignatureShare.maxSigners", maxSigners)
		fmt.Println(i, "VerifySignatureShare.verificationKey", hex.EncodeToString(verificationKey.Encode()))
		fmt.Println(i, "VerifySignatureShare.commitments", hex.EncodeToString(commitments.Encode()))
		fmt.Println(i, "VerifySignatureShare.message", hex.EncodeToString(message))
		fmt.Println(i, "VerifySignatureShare.signatureShare", hex.EncodeToString(signatureShare.Encode()))

		if err := configuration.VerifySignatureShare(signatureShare, message, commitments); err != nil {
			panic(
				fmt.Sprintf(
					"participant %v produced an invalid signature share: %s",
					signatureShare.SignerIdentifier,
					err,
				),
			)
		}
	}

	fmt.Println("Signature is valid.")

	// Output: Signature is valid.
}
