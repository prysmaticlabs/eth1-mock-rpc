package eth1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const depositContractABI = "[{\"name\":\"DepositEvent\",\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"amount\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"signature\",\"indexed\":false},{\"type\":\"bytes\",\"name\":\"index\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"outputs\":[],\"inputs\":[{\"type\":\"uint256\",\"name\":\"minDeposit\"},{\"type\":\"address\",\"name\":\"_drain_address\"}],\"constant\":false,\"payable\":false,\"type\":\"constructor\"},{\"name\":\"get_hash_tree_root\",\"outputs\":[{\"type\":\"bytes32\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":91734},{\"name\":\"get_deposit_count\",\"outputs\":[{\"type\":\"bytes\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":10493},{\"name\":\"deposit\",\"outputs\":[],\"inputs\":[{\"type\":\"bytes\",\"name\":\"pubkey\"},{\"type\":\"bytes\",\"name\":\"withdrawal_credentials\"},{\"type\":\"bytes\",\"name\":\"signature\"}],\"constant\":false,\"payable\":true,\"type\":\"function\",\"gas\":1334707},{\"name\":\"drain\",\"outputs\":[],\"inputs\":[],\"constant\":false,\"payable\":false,\"type\":\"function\",\"gas\":35823},{\"name\":\"MIN_DEPOSIT_AMOUNT\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":663},{\"name\":\"deposit_count\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":693},{\"name\":\"drain_address\",\"outputs\":[{\"type\":\"address\",\"name\":\"out\"}],\"inputs\":[],\"constant\":true,\"payable\":false,\"type\":\"function\",\"gas\":723}]"

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
