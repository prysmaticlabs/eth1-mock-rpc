package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/profile"
	"github.com/prysmaticlabs/eth1-mock-rpc/eth1"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/net/websocket"
)

const (
	maxRequestContentLength = 1024 * 512
	defaultErrorCode        = -32000
	eth1BlockTime           = time.Second * 10
	startingBlockNumber     = 2000
)

var (
	wsPort             = flag.String("ws-port", "7778", "Port on which to serve websocket listeners")
	httpPort           = flag.String("http-port", "7777", "Port on which to serve http listeners")
	host               = flag.String("host", "localhost", "Host on which to listen (default: localhost)")
	numGenesisDeposits = flag.Int("genesis-deposits", 0, "Number of deposits to read from the keystore to trigger the genesis event")
	blockTime          = flag.Int("block-time", 14, "Average time between blocks in seconds, default: 14s (Goerli testnet)")
	verbosity          = flag.String("verbosity", "info", "Logging verbosity (debug, info=default, warn, error, fatal, panic)")
	pprof              = flag.Bool("pprof", false, "Enable pprof")
	unencryptedKeysDir = flag.String("unencrypted-keys-dir", "", "Path to directory of json files containing unencrypted validator private keys")
	log                = logrus.WithField("prefix", "main")
	// use this flag when running non-interactively
	// otherwise, prompt will spam stdout
	promptForDeposits  = flag.Bool("prompt-for-deposits", true, "Prompt user to trigger deposits")
)

type server struct {
	depositsLock           sync.Mutex
	numDepositsReadyToSend int
	deposits               []*eth1.DepositData
	eth1BlocksByNumber     map[uint64]*types.Header
	eth1BlockNumbersByHash map[common.Hash]uint64
	eth1Logs               []types.Log
	eth1BlockNum           uint64
	depositsToSend         int
	eth1HeadFeed           *event.Feed
	genesisTime            uint64
}

type websocketHandler struct {
	blockNum      uint64
	close         chan bool
	readOperation chan []*jsonrpcMessage // Channel for read messages from the codec.
	readErr       chan error
}

func main() {
	flag.Parse()
	formatter := new(prefixed.TextFormatter)
	formatter.TimestampFormat = "2006-01-02 15:04:05"
	formatter.FullTimestamp = true
	logrus.SetFormatter(formatter)
	level, err := logrus.ParseLevel(*verbosity)
	if err != nil {
		log.Fatal(err)
	}
	logrus.SetLevel(level)

	if *numGenesisDeposits == 0 {
		log.Fatal("Please enter a valid number of --genesis-deposits to read from the keystore")
	}

	// If an unencrypted keys directory is not specified, we throw an error
	providedUnencryptedKeys := *unencryptedKeysDir != ""
	if !providedUnencryptedKeys {
		log.Fatal("Please enter a path to a directory of unencrypted private key JSON files for launching the mock server")
	}
	fileInfo, err := ioutil.ReadDir(*unencryptedKeysDir)
	if err != nil {
		log.Fatal(err)
	}
	validatorKeys := make([][]byte, 0)
	withdrawalKeys := make([][]byte, 0)
	for _, file := range fileInfo {
		r, err := os.Open(path.Join(*unencryptedKeysDir, file.Name()))
		if err != nil {
			log.Fatal(err)
		}
		vkey, wkey, err := parseUnencryptedKeysFile(r)
		if err != nil {
			log.Fatal(err)
		}
		validatorKeys = append(validatorKeys, vkey...)
		withdrawalKeys = append(withdrawalKeys, wkey...)
	}
	allDeposits, err := createDepositDataFromKeys(validatorKeys, withdrawalKeys)
	if err != nil {
		log.Fatal(err)
	}
	if *numGenesisDeposits > len(allDeposits) {
		log.Fatalf(
			"Number of --genesis-deposits %d > number of deposits found in keystore directory %d",
			*numGenesisDeposits,
			allDeposits,
		)
	}

	httpListener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", *host, *httpPort))
	if err != nil {
		log.Fatal(err)
	}
	wsListener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", *host, *wsPort))
	if err != nil {
		log.Fatal(err)
	}

	// We also compute a history of eth1 blocks to be used to respond to RPC requests for
	// blocks by number, getting our mock server to closely resemble a real chain.
	currentBlockNumber := uint64(startingBlockNumber)
	blocksByNumber := eth1.ConstructBlocksByNumber(currentBlockNumber, eth1BlockTime)
	blockNumbersByHash := make(map[common.Hash]uint64)
	for k, v := range blocksByNumber {
		h := v.Hash()
		blockNumbersByHash[h] = k
	}

	// We precalculate a list of deposit logs from the entire in-memory deposits list.
	logs, err := eth1.DepositEventLogs(allDeposits)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < *numGenesisDeposits; i++ {
		logs[i].BlockHash = blocksByNumber[currentBlockNumber].Hash()
		logs[i].BlockNumber = currentBlockNumber
	}

	srv := &server{
		numDepositsReadyToSend: *numGenesisDeposits,
		deposits:               allDeposits,
		eth1Logs:               logs,
		eth1BlockNum:           currentBlockNumber,
		eth1BlockNumbersByHash: blockNumbersByHash,
		eth1BlocksByNumber:     blocksByNumber,
		eth1HeadFeed:           new(event.Feed),
		genesisTime:            uint64(time.Now().Add(10 * time.Second).Unix()),
	}

	if *pprof {
		defer profile.Start().Stop()
	}

	log.Println("Starting HTTP listener on port :7777")
	go http.Serve(httpListener, srv)

	log.Println("Starting WebSocket listener on port :7778")
	wsSrv := &http.Server{Handler: srv.ServeWebsocket()}
	go wsSrv.Serve(wsListener)

	if *promptForDeposits {
		go srv.listenForDepositTrigger()
	}

	go srv.advanceEth1Chain(*blockTime)

	select {}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	body := io.LimitReader(r.Body, maxRequestContentLength)
	conn := &httpServerConn{Reader: body, Writer: w, r: r}
	codec := NewJSONCodec(conn)
	defer codec.Close()
	msgs, batch, err := codec.Read()
	if err != nil {
		log.WithError(err).Error("Could not read data from request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	requestItem := msgs[0]
	if !requestItem.isCall() {
		log.WithField("messageType", requestItem.Method).Error("Can only serve RPC call types via HTTP")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stringRep := requestItem.String()
	switch requestItem.Method {
	case "eth_getBlockByNumber":
		blocks := make([]*types.Header, 0)
		for i := 0; i < len(msgs); i++ {
			typs := []reflect.Type{
				reflect.TypeOf("s"),
				reflect.TypeOf(true),
			}
			args, err := parsePositionalArguments(requestItem.Params, typs)
			if err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			var block *types.Header
			var ok bool
			if args[0].String() == "latest" {
				block, ok = s.eth1BlocksByNumber[s.eth1BlockNum]
				if !ok {
					log.Errorf("Block with 'latest' does not exist at blocknumber %d", s.eth1BlockNum)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			} else {
				num, err := hexutil.DecodeBig(args[0].String())
				if err != nil {
					log.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
				}
				if num.Uint64() < startingBlockNumber {
					num = num.SetInt64(startingBlockNumber)
				}
				block, ok = s.eth1BlocksByNumber[num.Uint64()]
				if !ok {
					log.Errorf("Block %d does not exist", num.Uint64())
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			blocks = append(blocks, block)
		}
		if len(blocks) == 1 && !batch {
			response := requestItem.response(blocks[0])
			if err := codec.Write(ctx, response); err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		responses := make([]*jsonrpcMessage, 0)
		for i, b := range blocks {
			res := msgs[i].response(b)
			responses = append(responses, res)
		}
		if err := codec.Write(ctx, responses); err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case "eth_getBlockByHash":
		typs := []reflect.Type{
			reflect.TypeOf("s"),
			reflect.TypeOf(true),
		}
		args, err := parsePositionalArguments(requestItem.Params, typs)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		h := args[0].String()
		// Strip out the 0x prefix.
		blockHashBytes, err := hex.DecodeString(h[2:])
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		var blockHash [32]byte
		copy(blockHash[:], blockHashBytes)
		numByHash := s.eth1BlockNumbersByHash[blockHash]
		if numByHash < startingBlockNumber {
			numByHash = startingBlockNumber
		}
		block := s.eth1BlocksByNumber[numByHash]
		response := requestItem.response(block)
		if err := codec.Write(ctx, response); err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case "eth_getLogs":
		response := requestItem.response(s.eth1Logs[:s.numDepositsReadyToSend])
		if err := codec.Write(ctx, response); err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	case "eth_call":
		if strings.Contains(stringRep, eth1.DepositMethodID()) {
			count := eth1.DepositCount(s.deposits[:s.numDepositsReadyToSend])
			depCount, err := eth1.PackDepositCount(count[:])
			if err != nil {
				log.WithError(err).Error("Could not respond to HTTP request")
				w.WriteHeader(http.StatusInternalServerError)
			}
			response := requestItem.response(fmt.Sprintf("%#x", depCount))
			if err := codec.Write(ctx, response); err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		if strings.Contains(stringRep, eth1.DepositLogsID()) {
			root, err := eth1.DepositRoot(s.deposits[:s.numDepositsReadyToSend])
			if err != nil {
				log.WithError(err).Error("Could not respond to HTTP request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			response := requestItem.response(fmt.Sprintf("%#x", root))
			if err := codec.Write(ctx, response); err != nil {
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		s.defaultResponse(w)
	default:
		s.defaultResponse(w)
	}
}

func (s *server) defaultResponse(w http.ResponseWriter) {
	log.Error("Could not respond to HTTP request")
	w.WriteHeader(http.StatusBadRequest)
}

func (s *server) ServeWebsocket() http.Handler {
	return websocket.Server{
		Handler: func(conn *websocket.Conn) {
			codec := newWebsocketCodec(conn)
			wsHandler := &websocketHandler{
				blockNum:      0,
				close:         make(chan bool),
				readOperation: make(chan []*jsonrpcMessage),
				readErr:       make(chan error),
			}

			defer codec.Close()
			// Listen to read events from the codec and dispatch events or errors accordingly.
			go wsHandler.websocketReadLoop(codec)
			go wsHandler.dispatchWebsocketEventLoop(codec, s.eth1HeadFeed)
			<-codec.Closed()
		},
	}
}

func (w *websocketHandler) dispatchWebsocketEventLoop(codec ServerCodec, headFeed *event.Feed) {
	var latestSubID rpc.ID
	headChan := make(chan *types.Header, 1)
	sub := headFeed.Subscribe(headChan)
	defer sub.Unsubscribe()
	for {
		select {
		case <-w.close:
			return
		case err := <-w.readErr:
			log.WithError(err).Error("Could not read data from request")
			return
		case head := <-headChan:
			data, _ := json.Marshal(head)
			params, _ := json.Marshal(&subscriptionResult{ID: string(latestSubID), Result: data})
			ctx := context.Background()
			item := &jsonrpcMessage{
				Version: "2.0",
				Method:  "eth_subscription",
				Params:  params,
			}
			if err := codec.Write(ctx, item); err != nil {
				log.Error(err)
				continue
			}
		case msgs := <-w.readOperation:
			sub := &rpc.Subscription{ID: rpc.NewID()}
			item := &jsonrpcMessage{
				Version: msgs[0].Version,
				ID:      msgs[0].ID,
			}
			latestSubID = sub.ID
			newItem := item.response(sub)
			if err := codec.Write(context.Background(), newItem); err != nil {
				log.Error(err)
				continue
			}
		}
	}
}

func (w *websocketHandler) websocketReadLoop(codec ServerCodec) {
	for {
		select {
		case <-w.close:
			return
		default:
			msgs, _, err := codec.Read()
			if _, ok := err.(*json.SyntaxError); ok {
				if err := codec.Write(context.Background(), errorMessage(err)); err != nil {
					log.Error(err)
					continue
				}
			}
			if err != nil {
				w.readErr <- err
				return
			}
			w.readOperation <- msgs
		}
	}
}

func (s *server) listenForDepositTrigger() {
	reader := bufio.NewReader(os.Stdin)
	for {
		maxAllowed := len(s.deposits) - s.numDepositsReadyToSend
		log.Printf(
			"Enter the number of new eth2 deposits to trigger (max allowed %d): ",
			maxAllowed,
		)
		fmt.Print(">> ")
		line, _, err := reader.ReadLine()
		if err != nil {
			log.Error(err)
			continue
		}
		num, err := strconv.Atoi(string(line))
		if err != nil {
			log.Error(err)
		}
		if num > maxAllowed {
			log.Errorf(
				"You have already sent %d/%d available deposits in keystore, cannot submit more",
				len(s.deposits),
				s.numDepositsReadyToSend,
			)
			continue
		}
		s.depositsToSend = num
		for s.depositsToSend != 0 {
			time.Sleep(1 * time.Second)
			// wait till it's sent again
		}
	}
}

func (s *server) advanceEth1Chain(blockTime int) {
	tick := time.NewTicker(time.Second * time.Duration(blockTime))
	for {
		select {
		case <-tick.C:
			s.eth1BlockNum++
			head := eth1.BlockHeader(s.eth1BlockNum)
			s.eth1BlocksByNumber[s.eth1BlockNum] = head
			s.eth1BlockNumbersByHash[head.Hash()] = s.eth1BlockNum
			for i := s.numDepositsReadyToSend; i < (s.numDepositsReadyToSend + s.depositsToSend); i++ {
				s.eth1Logs[i].BlockHash = s.eth1BlocksByNumber[s.eth1BlockNum].Hash()
				s.eth1Logs[i].BlockNumber = s.eth1BlockNum
			}
			s.numDepositsReadyToSend += s.depositsToSend
			s.depositsToSend = 0
			s.eth1HeadFeed.Send(head)
		}
	}
}
