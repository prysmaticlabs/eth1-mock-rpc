package eth1

import (
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

var (
	// MaxEffectiveBalance of an active eth2 validator.
	MaxEffectiveBalance      = uint64(3.2 * 1e9)
	blsWithdrawalPrefixByte  = byte(0)
	domainDeposit            = [4]byte{3, 0, 0, 0}
	genesisForkVersion       = []byte{0, 0, 0, 0}
	depositContractTreeDepth = uint64(32)
	depositEventSignature    = []byte("DepositEvent(bytes,bytes,bytes,bytes,bytes)")
)

// DepositData defines an Ethereum 2.0 data structure used as part of the
// core protocol state transition - onboarding new active validators
// into the chain accordingly.
type DepositData struct {
	Pubkey                []byte `json:"pubkey,omitempty" ssz-size:"48"`
	WithdrawalCredentials []byte `json:"withdrawal_credentials,omitempty" ssz-size:"32"`
	Amount                uint64 `json:"amount,omitempty"`
	Signature             []byte `json:"signature,omitempty" ssz-size:"96"`
}

// CreateDepositData takes in raw private key bytes and a deposit amount and generates
// the proper DepositData Eth2 struct type. This involves BLS signing the deposit,
// generating hashed withdrawal credentials, and including the public key from the validator's
// private key into the deposit struct.
func CreateDepositData(validatorKey []byte, withdrawalKey []byte, amountInGwei uint64) (*DepositData, error) {
	sk1, err := bls.SecretKeyFromBytes(validatorKey)
	if err != nil {
		return nil, err
	}
	sk2, err := bls.SecretKeyFromBytes(withdrawalKey)
	if err != nil {
		return nil, err
	}
	di := &DepositData{
		Pubkey:                sk1.PublicKey().Marshal(),
		WithdrawalCredentials: withdrawalCredentialsHash(sk2),
		Amount:                amountInGwei,
	}

	sr, err := ssz.SigningRoot(di)
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
