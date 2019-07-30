package eth1

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// Handler provides methods for handling eth1 JSON-RPC requests using
// mock or constructed data accordingly.
type Handler struct {
	Deposits    []*DepositData
	GenesisTime uint64
}

// DepositRoot produces a hash tree root of a list of deposits
// to match the output of the deposit contract on the eth1 chain.
func (h *Handler) DepositRoot() ([32]byte, error) {
	return ssz.HashTreeRootWithCapacity(h.Deposits, 1<<depositContractTreeDepth)
}

func (h *Handler) DepositMethodID() string {
	methodHash := hashutil.HashKeccak256([]byte("get_deposit_count()"))
	dataField := "data\":\"0x" + common.Bytes2Hex(methodHash[:4]) + "\""
	return dataField
}

func (h *Handler) DepositLogsID() string {
	// TODO():Find the proper way to retrieve the hash
	eventHash := "863a311b"
	dataField := "data\":\"0x" + eventHash + "\""
	return dataField
}

func (h *Handler) DepositCount() [8]byte {
	count := uint64(len(h.Deposits))
	var depCount [8]byte
	binary.LittleEndian.PutUint64(depCount[:], count)
	return depCount
}

// LatestChainHead returns the latest eth1 chain into a channel.
// TODO: Convert into a channel push.
func (h *Handler) LatestChainHead() *types.Header {
	head := &types.Header{
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
	return head
}

// BlockHeaderByHash returns a block header given a raw hash.
func (h *Handler) BlockHeaderByHash() *types.Header {
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
		Time:        uint64(h.GenesisTime),
		Extra:       []byte("hello world"),
	}
}

// BlockHeaderByNumber returns a block header given a block height.
func (h *Handler) BlockHeaderByNumber() *types.Header {
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
func (h *Handler) DepositEventLogs() ([]types.Log, error) {
	depositEventHash := hashutil.HashKeccak256(depositEventSignature)
	logs := make([]types.Log, len(h.Deposits))
	for i := 0; i < len(logs); i++ {
		indexBuf := make([]byte, 8)
		amountBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(amountBuf, h.Deposits[i].Amount)
		binary.LittleEndian.PutUint64(indexBuf, uint64(i))
		depositLog, err := packDepositLog(
			h.Deposits[i].Pubkey,
			h.Deposits[i].WithdrawalCredentials,
			amountBuf,
			h.Deposits[i].Signature,
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
