package eth1

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// ResponseWriter details a struct which can write arbitrary data somewhere
// given a context and and a value.
type ResponseWriter interface {
	Write(context.Context, interface{}) error
}

// Handler provides methods for handling eth1 JSON-RPC requests using
// mock or constructed data accordingly.
type Handler struct {
	Deposits []*depositData
	Writer   ResponseWriter
}

func (h *Handler) handleDepositRoot(msg *jsonrpcMessage) error {
	depositRoot, err := ssz.HashTreeRootWithCapacity(h.Deposits, 1<<depositContractTreeDepth)
	if err != nil {
		return err
	}
	newItem := msg.response(fmt.Sprintf("%#x", depositRoot))
	h.Writer.Write(context.Background(), newItem)
	return nil
}

func (h *Handler) handleBlockByHash(msg *jsonrpcMessage) {
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
	newItem := msg.response(head)
	h.Writer.Write(context.Background(), newItem)
}

func (h *Handler) handleBlockByNumber(msg *jsonrpcMessage) {
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
	newItem := msg.response(head)
	h.Writer.Write(context.Background(), newItem)
}

func (h *Handler) handleGetLogs(msg *jsonrpcMessage) error {
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
			return nil
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
	newItem := msg.response(logs)
	h.Writer.Write(context.Background(), newItem)
	return nil
}
