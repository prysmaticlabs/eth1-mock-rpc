package main

import (
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var (
	blsWithdrawalPrefixByte  = byte(0)
	domainDeposit            = [4]byte{3, 0, 0, 0}
	genesisForkVersion       = []byte{0, 0, 0, 0}
	maxEffectiveBalance      = uint64(3.2 * 1e9)
	depositContractTreeDepth = uint64(32)
)

type keysContainer struct {
	Keys []*keys `json:"keys"`
}

type keys struct {
	ValidatorKey  []byte `json:"validator_key" ssz-size:"32"`
	WithdrawalKey []byte `json:"withdrawal_key" ssz-size:"32"`
}

type depositData struct {
	Pubkey                []byte `json:"pubkey,omitempty" ssz-size:"48"`
	WithdrawalCredentials []byte `json:"withdrawal_credentials,omitempty" ssz-size:"32"`
	Amount                uint64 `json:"amount,omitempty"`
	Signature             []byte `json:"signature,omitempty" ssz-size:"96"`
}

func createDepositData(validatorKey []byte, withdrawalKey []byte, amountInGwei uint64) (*depositData, error) {
	sk1, err := bls.SecretKeyFromBytes(validatorKey)
	if err != nil {
		return nil, err
	}
	sk2, err := bls.SecretKeyFromBytes(withdrawalKey)
	if err != nil {
		return nil, err
	}
	di := &depositData{
		Pubkey:                sk1.PublicKey().Marshal(),
		WithdrawalCredentials: withdrawalCredentialsHash(sk2),
		Amount:                amountInGwei,
	}

	sr, err := ssz.HashTreeRoot(di)
	if err != nil {
		return nil, err
	}

	domain := bls.Domain(domainDeposit[:], genesisForkVersion)
	di.Signature = sk1.Sign(sr[:], domain).Marshal()
	return di, nil
}

// withdrawalCredentialsHash forms a 32 byte hash of the withdrawal public
// address.
//
// The specification is as follows:
//   withdrawal_credentials[:1] == BLS_WITHDRAWAL_PREFIX_BYTE
//   withdrawal_credentials[1:] == hash(withdrawal_pubkey)[1:]
// where withdrawal_credentials is of type bytes32.
func withdrawalCredentialsHash(withdrawalKey *bls.SecretKey) []byte {
	h := hashutil.HashKeccak256(withdrawalKey.PublicKey().Marshal())
	return append([]byte{blsWithdrawalPrefixByte}, h[0:]...)[:32]
}
