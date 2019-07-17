package eth1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

const depositContractABI = "[{\"name\":\"DepositEvent\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"amount\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"signature\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"index\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"outputs\":[],\"inputs\":[{\"type\":\"uint256\",\"name\":\"minDeposit\"},{\"type\":\"address\",\"name\":\"_drain_address\"}],\"constant\":false,\"payable\":false,\"type\":\"constructor\"},{\"name\":\"get_hash_tree_root\",\"outputs\":[{\"type\":\"bytes32\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":91734},{\"name\":\"get_deposit_count\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":10493},{\"name\":\"deposit\",\"outputs\":[],\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\"},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\"},{\"type\":\"bytes\",\"name\":\"signature\"}],\"constant\":false,\"payable\":true,\"type\":\"function\",\"gas\":1334707},{\"name\":\"drain\",\"outputs\":[],\"inputs\":[],\"constant\":false,\"payable\":false,\"type\":\"function\",\"gas\":35823},{\"name\":\"MIN_DEPOSIT_AMOUNT\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":663},{\"name\":\"deposit_count\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":693},{\"name\":\"drain_address\",\"outputs\":[{\"type\":\"address\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":723}]"

var (
	// MaxEffectiveBalance of an active eth2 validator.
	MaxEffectiveBalance      = uint64(3.2 * 1e9)
	blsWithdrawalPrefixByte  = byte(0)
	domainDeposit            = [4]byte{3, 0, 0, 0}
	genesisForkVersion       = []byte{0, 0, 0, 0}
	depositContractTreeDepth = uint64(32)
	depositEventSignature    = []byte("DepositEvent(bytes,bytes,bytes,bytes,bytes)")
)

type keysContainer struct {
	Keys []*keys `json:"keys"`
}

type keys struct {
	ValidatorKey  []byte `json:"validator_key" ssz-size:"32"`
	WithdrawalKey []byte `json:"withdrawal_key" ssz-size:"32"`
}

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

// packDepositLog uses the deposit contract ABI to pack raw information
// into an encoded set of bytes to be included in the Data field of a
// types.Log item as specified from the go-ethereum repository.
func packDepositLog(
	pubkey []byte,
	withdrawalCredentials []byte,
	amount []byte,
	signature []byte,
	index []byte,
) ([]byte, error) {
	reader := bytes.NewReader([]byte(depositContractABI))
	contractAbi, err := abi.JSON(reader)
	if err != nil {
		return nil, err
	}
	return contractAbi.Events["DepositEvent"].Inputs.Pack(pubkey, withdrawalCredentials, amount, signature, index)
}
