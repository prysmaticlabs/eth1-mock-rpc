package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/net/websocket"
)

const (
	maxRequestContentLength = 1024 * 512
	defaultErrorCode        = -32000
)

var (
	keystorePath          = flag.String("keystore-path", "", "Path to a validator keystore directory")
	password              = flag.String("password", "", "Password to unlocking the validator keystore directory")
	wsPort                = flag.String("ws-port", "7778", "Port on which to serve websocket listeners")
	httpPort              = flag.String("http-port", "7777", "Port on which to serve http listeners")
	depositEventSignature = []byte("DepositEvent(bytes,bytes,bytes,bytes,bytes)")
	log                   = logrus.WithField("prefix", "main")
)

type server struct {
	deposits      []*depositData
	close         chan struct{}
	readOperation chan []*jsonrpcMessage // read messages.
	readErr       chan error             // errors from read.
}

func main() {
	flag.Parse()
	formatter := new(prefixed.TextFormatter)
	formatter.TimestampFormat = "2006-01-02 15:04:05"
	formatter.FullTimestamp = true
	logrus.SetFormatter(formatter)

	// deposits, err := createDepositDataFromKeystore(*keystorePath, *password)
	// if err != nil {
	// 	log.Fatalf("Could not create deposit data from keystore directory: %v", err)
	// }
	// log.Infof("Successfully loaded %d deposits from the keystore directory", len(deposits))
	httpListener, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", *httpPort))
	if err != nil {
		panic(err)
	}
	wsListener, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", *wsPort))
	if err != nil {
		panic(err)
	}
	srv := &server{
		deposits:      nil,
		close:         make(chan struct{}),
		readOperation: make(chan []*jsonrpcMessage),
		readErr:       make(chan error),
	}
	log.Println("Starting HTTP listener on port :7777")
	go http.Serve(httpListener, srv)

	log.Println("Starting WebSocket listener on port :7778")
	wsSrv := &http.Server{Handler: srv.WebsocketHandler()}
	go wsSrv.Serve(wsListener)

	select {}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body := io.LimitReader(r.Body, maxRequestContentLength)
	conn := &httpServerConn{Reader: body, Writer: w, r: r}
	codec := NewJSONCodec(conn)
	defer codec.Close()
	msgs, _, err := codec.Read()
	if err != nil {
		log.WithError(err).Error("Could not read data from request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	item := msgs[0]
	if !item.isCall() {
		log.WithField("messageType", item.Method).Error("Can only serve RPC call types via HTTP")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.WithField("method", item.Method).Info("Received HTTP-RPC request")
	log.Infof("%v", item)
	if item.Method == "eth_getBlockByNumber" {
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
			Time:        10,
			Extra:       []byte("hello world"),
		}
		newItem := item.response(head)
		codec.Write(context.Background(), newItem)
	} else if item.Method == "eth_getLogs" {
		depositEventHash := hashutil.HashKeccak256(depositEventSignature)
		logs := make([]types.Log, len(s.deposits))
		for i := 0; i < len(logs); i++ {
			indexBuf := make([]byte, 8)
			amountBuf := make([]byte, 8)
			binary.LittleEndian.PutUint64(amountBuf, s.deposits[i].Amount)
			binary.LittleEndian.PutUint64(indexBuf, uint64(i))
			depositLog, err := packDepositLog(
				s.deposits[i].Pubkey,
				s.deposits[i].WithdrawalCredentials,
				amountBuf,
				s.deposits[i].Signature,
				indexBuf,
			)
			if err != nil {
				panic(err)
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
		newItem := item.response(logs)
		codec.Write(context.Background(), newItem)
	} else if item.Method == "eth_getBlockByHash" {
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
		newItem := item.response(head)
		codec.Write(context.Background(), newItem)
	} else {
		depositRoot, err := ssz.HashTreeRootWithCapacity(s.deposits, 1<<depositContractTreeDepth)
		if err != nil {
			panic(err)
		}
		newItem := item.response(fmt.Sprintf("%#x", depositRoot))
		codec.Write(context.Background(), newItem)

	}
}

func (s *server) WebsocketHandler() http.Handler {
	return websocket.Server{
		Handler: func(conn *websocket.Conn) {
			codec := newWebsocketCodec(conn)
			defer codec.Close()
			// Listen to read events from the codec
			// and dispatch events or errors accordingly.
			go s.read(codec)
			go s.dispatch(codec)
			<-codec.Closed()
		},
	}
}

func (s *server) dispatch(codec ServerCodec) {
	tick := time.NewTicker(time.Second * 10)
	var latestSubID rpc.ID
	for {
		select {
		case <-s.close:
			return
		case err := <-s.readErr:
			log.WithError(err).Error("Could not read")
		case <-tick.C:
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
			data, _ := json.Marshal(head)
			params, _ := json.Marshal(&subscriptionResult{ID: string(latestSubID), Result: data})
			ctx := context.Background()
			item := &jsonrpcMessage{
				Version: "2.0",
				Method:  "eth_subscription",
				Params:  params,
			}
			log.Infof("Writing item: %v", item)
			codec.Write(ctx, item)
		case msgs := <-s.readOperation:
			sub := &rpc.Subscription{ID: rpc.NewID()}
			item := &jsonrpcMessage{
				Version: msgs[0].Version,
				ID:      msgs[0].ID,
			}
			latestSubID = sub.ID
			newItem := item.response(sub)
			codec.Write(context.Background(), newItem)
		}
	}
}

func (s *server) read(codec ServerCodec) {
	for {
		msgs, _, err := codec.Read()
		if _, ok := err.(*json.SyntaxError); ok {
			codec.Write(context.Background(), errorMessage(err))
		}
		if err != nil {
			s.readErr <- err
			return
		}
		s.readOperation <- msgs
	}
}
