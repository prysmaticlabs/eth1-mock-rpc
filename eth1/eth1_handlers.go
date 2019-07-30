package eth1

import (
	"encoding/binary"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// DepositRoot produces a hash tree root of a list of deposits
// to match the output of the deposit contract on the eth1 chain.
func DepositRoot(deposits []*DepositData) ([32]byte, error) {
	return ssz.HashTreeRootWithCapacity(deposits, 1<<depositContractTreeDepth)
}

// DepositMethodID returns the ABI encoded method value as a hex string.
func DepositMethodID() string {
	methodHash := hashutil.HashKeccak256([]byte("get_deposit_count()"))
	return "data\":\"0x" + common.Bytes2Hex(methodHash[:4]) + "\""
}

// DepositLogsID returns the event hash from the ABI corresponding to
// fetching the deposit logs event.
func DepositLogsID() string {
	// TODO():Find the proper way to retrieve the hash
	eventHash := "863a311b"
	return "data\":\"0x" + eventHash + "\""
}

// DepositCount returns an encoded number of deposits.
func DepositCount(deposits []*DepositData) [8]byte {
	count := uint64(len(deposits))
	var depCount [8]byte
	binary.LittleEndian.PutUint64(depCount[:], count)
	return depCount
}

// LatestChainHead returns the latest eth1 chain into a channel.
func LatestChainHead(blockNum uint64) *types.Header {
	head := &types.Header{
		ParentHash:  common.Hash([32]byte{}),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address([20]byte{}),
		Root:        common.Hash([32]byte{}),
		TxHash:      types.EmptyRootHash,
		ReceiptHash: common.Hash([32]byte{}),
		Bloom:       types.Bloom{},
		Difficulty:  big.NewInt(20),
		Number:      big.NewInt(int64(blockNum)),
		GasLimit:    100,
		GasUsed:     100,
		Time:        uint64(time.Now().Unix()),
		Extra:       []byte("hello world"),
	}
	return head
}

// BlockHeaderByHash returns a block header given a raw hash.
func BlockHeaderByHash(genesisTime uint64) *types.Header {
	return &types.Header{
		ParentHash:  common.Hash([32]byte{}),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address([20]byte{}),
		Root:        common.Hash([32]byte{}),
		TxHash:      types.EmptyRootHash,
		ReceiptHash: common.Hash([32]byte{}),
		Bloom:       types.Bloom{},
		Difficulty:  big.NewInt(20),
		Number:      big.NewInt(int64(100)),
		GasLimit:    100,
		GasUsed:     100,
		Time:        genesisTime,
		Extra:       []byte("hello world"),
	}
}

// BlockHeaderByNumber returns a block header given a block height.
func BlockHeaderByNumber() *types.Header {
	return &types.Header{
		ParentHash:  common.Hash([32]byte{}),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address([20]byte{}),
		Root:        common.Hash([32]byte{}),
		TxHash:      types.EmptyRootHash,
		ReceiptHash: common.Hash([32]byte{}),
		Bloom:       types.Bloom{},
		Difficulty:  big.NewInt(20),
		Number:      big.NewInt(int64(100)),
		GasLimit:    100,
		GasUsed:     100,
		Time:        1578009600,
		Extra:       []byte("hello world"),
	}
}

// DepositEventLogs returns a list of eth1 logs that have occurred
// at a deposit contract address. This uses an internal list of deposit data
// to return instead of relying on a real network and parsing a real deposit contract
// for this information.
func DepositEventLogs(deposits []*DepositData) ([]types.Log, error) {
	depositEventHash := hashutil.HashKeccak256(depositEventSignature)
	logs := make([]types.Log, len(deposits))
	for i := 0; i < len(logs); i++ {
		indexBuf := make([]byte, 8)
		amountBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(amountBuf, deposits[i].Amount)
		binary.LittleEndian.PutUint64(indexBuf, uint64(i))
		depositLog, err := packDepositLog(
			deposits[i].Pubkey,
			deposits[i].WithdrawalCredentials,
			amountBuf,
			deposits[i].Signature,
			indexBuf,
		)
		if err != nil {
			return nil, nil
		}
		logs[i] = types.Log{
			BlockHash: common.Hash([32]byte{1, 2, 3, 4, 5}),
			Address:   common.Address([20]byte{}),
			Topics:    []common.Hash{depositEventHash},
			Data:      depositLog,
			TxHash:    common.Hash([32]byte{}),
			TxIndex:   100,
			Index:     10,
		}
	}
	return logs, nil
}
