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
	methodHash := hashutil.HashKeccak256([]byte("get_deposit_root()"))
	return "data\":\"0x" + common.Bytes2Hex(methodHash[:4]) + "\""
}

// DepositCount returns an encoded number of deposits.
func DepositCount(deposits []*DepositData) [8]byte {
	count := uint64(len(deposits))
	var depCount [8]byte
	binary.LittleEndian.PutUint64(depCount[:], count)
	return depCount
}

// ConstructBlocksByNumber builds a list of historical blocks down to block 0 from some current block number,
// used to simulate a real eth1 chain which can be queried for all of its blocks by their respective number.
func ConstructBlocksByNumber(currentBlockNum uint64, blockTime time.Duration) map[uint64]*types.Header {
	m := make(map[uint64]*types.Header)
	currentTime := uint64(time.Now().Unix())
	for i := currentBlockNum; i > 0; i-- {
		header := &types.Header{
			ParentHash:  common.Hash([32]byte{}),
			UncleHash:   types.EmptyUncleHash,
			Coinbase:    common.Address([20]byte{}),
			Root:        common.Hash([32]byte{}),
			TxHash:      types.EmptyRootHash,
			ReceiptHash: common.Hash([32]byte{}),
			Bloom:       types.Bloom{},
			Difficulty:  big.NewInt(20),
			Number:      big.NewInt(int64(i)),
			GasLimit:    100,
			GasUsed:     100,
			Time:        currentTime,
			Extra:       []byte("hello world"),
		}
		m[i] = header
		currentTime = currentTime - uint64(blockTime.Seconds())
	}
	return m
}

// BlockHeader returns a block header with time.Now and a blockNum.
func BlockHeader(blockNum uint64) *types.Header {
	return &types.Header{
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
			Address: common.Address([20]byte{}),
			Topics:  []common.Hash{depositEventHash},
			Data:    depositLog,
			TxHash:  common.Hash([32]byte{}),
			TxIndex: 100,
			Index:   10,
		}
	}
	return logs, nil
}
