package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/eth1-mock-rpc/eth1"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"golang.org/x/net/websocket"
)

const (
	maxRequestContentLength = 1024 * 512
	defaultErrorCode        = -32000
)

var (
	keystorePath            = flag.String("keystore-path", "", "Path to a validator keystore directory")
	password                = flag.String("password", "", "Password to unlocking the validator keystore directory")
	wsPort                  = flag.String("ws-port", "7778", "Port on which to serve websocket listeners")
	httpPort                = flag.String("http-port", "7777", "Port on which to serve http listeners")
	log                     = logrus.WithField("prefix", "main")
	validatorDepositsOutput = flag.String("output", "", "Validator credentials and deposit output in YAML")
	persistedDepositsJSON   = "persistedDeposits.json"
)

type server struct {
	deposits []*eth1.DepositData
}

type websocketHandler struct {
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

	var deposits []*eth1.DepositData
	tmp := os.TempDir()
	cachePath := path.Join(tmp, persistedDepositsJSON)

	// We attempt to retrieve deposits from a local tmp file
	// as an optimization to prevent reading and decrypting raw private keys
	// from the validator keystore every single time the mock server is launched.
	if len(*validatorDepositsOutput) > 0 {
		log.Infof("Creating validator keys from %s, this may take a while...", *keystorePath)
		pubkeys, privkeys, err := createValidatorKeysFromKeystore(*keystorePath, *password)
		if err != nil {
			log.Fatalf("Could not create validator keys from keystore directory: %v", err)
		}
		log.Infof("Creating deposit data from %s, this may take a while...", *keystorePath)
		deposits, err = createDepositDataFromKeystore(*keystorePath, *password)
		if err != nil {
			log.Fatalf("Could not create deposit data from keystore directory: %v", err)
		}
		log.Infof(*validatorDepositsOutput)
		w, err := os.Create(*validatorDepositsOutput)
		if err != nil {
			log.Fatal(err)
		}
		if err := persistValidatorDepositData(w, pubkeys, privkeys, deposits); err != nil {
			log.Errorf("Could not persist deposits to disk: %v", err)
		}
		log.Infof("Successfully created file: %s", *validatorDepositsOutput)
		os.Exit(0)

	} else if r, err := os.Open(cachePath); err == nil {
		deposits, err = retrieveDepositData(r)
		if err != nil {
			log.Fatalf("Could not retrieve deposits from %s: %v", cachePath, err)
		}
	} else {
		// If the file does not exist at the tmp directory, we decrypt
		// from the keystore directory and then attempt to persist to the cache.
		log.Infof("Decrypting private keys from %s, this may take a while...", *keystorePath)
		deposits, err = createDepositDataFromKeystore(*keystorePath, *password)
		if err != nil {
			log.Fatalf("Could not create deposit data from keystore directory: %v", err)
		}
		w, err := os.Create(cachePath)
		if err != nil {
			log.Fatal(err)
		}
		if err := persistDepositData(w, deposits); err != nil {
			log.Errorf("Could not persist deposits to disk: %v", err)
		}
		//} else {
		//	log.Fatalf("Could not read from %s: %v", cachePath, err)
	}

	log.Infof("Successfully loaded %d deposits from the keystore directory", len(deposits))
	httpListener, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", *httpPort))
	if err != nil {
		panic(err)
	}
	wsListener, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", *wsPort))
	if err != nil {
		panic(err)
	}
	srv := &server{
		deposits: deposits,
	}
	log.Println("Starting HTTP listener on port :7777")
	go http.Serve(httpListener, srv)

	log.Println("Starting WebSocket listener on port :7778")
	wsSrv := &http.Server{Handler: srv.ServeWebsocket()}
	go wsSrv.Serve(wsListener)

	select {}
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
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
	requestItem := msgs[0]
	if !requestItem.isCall() {
		log.WithField("messageType", requestItem.Method).Error("Can only serve RPC call types via HTTP")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.WithField("method", requestItem.Method).Info("Received HTTP-RPC request")
	log.Infof("%v", requestItem)
	ethHandler := &eth1.Handler{
		Deposits: s.deposits,
	}

	switch requestItem.Method {
	case "eth_getBlockByNumber":
		block := ethHandler.BlockHeaderByNumber()
		response := requestItem.response(block)
		codec.Write(ctx, response)
	case "eth_getBlockByHash":
		block := ethHandler.BlockHeaderByHash()
		response := requestItem.response(block)
		codec.Write(ctx, response)
	case "eth_getLogs":
		logs, err := ethHandler.DepositEventLogs()
		if err != nil {
			log.WithError(err).Error("Could not respond to HTTP request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		response := requestItem.response(logs)
		codec.Write(ctx, response)
	default:
		// TODO: handle this by method name and use default for unknown cases.
		root, err := ethHandler.DepositRoot()
		if err != nil {
			log.WithError(err).Error("Could not respond to HTTP request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		response := requestItem.response(fmt.Sprintf("%#x", root))
		codec.Write(ctx, response)
	}
}

func (s *server) ServeWebsocket() http.Handler {
	return websocket.Server{
		Handler: func(conn *websocket.Conn) {
			codec := newWebsocketCodec(conn)
			wsHandler := &websocketHandler{
				close:         make(chan bool),
				readOperation: make(chan []*jsonrpcMessage),
				readErr:       make(chan error),
			}

			defer codec.Close()
			// Listen to read events from the codec and dispatch events or errors accordingly.
			go wsHandler.websocketReadLoop(codec)
			go wsHandler.dispatchWebsocketEventLoop(codec, s.deposits)
			<-codec.Closed()
		},
	}
}

func (w *websocketHandler) dispatchWebsocketEventLoop(codec ServerCodec, deposits []*eth1.DepositData) {
	eth := &eth1.Handler{
		Deposits: deposits,
	}
	tick := time.NewTicker(time.Second * 10)
	var latestSubID rpc.ID
	for {
		select {
		case <-w.close:
			return
		case err := <-w.readErr:
			log.WithError(err).Error("Could not read data from request")
			return
		case <-tick.C:
			head := eth.LatestChainHead()
			data, _ := json.Marshal(head)
			params, _ := json.Marshal(&subscriptionResult{ID: string(latestSubID), Result: data})
			ctx := context.Background()
			item := &jsonrpcMessage{
				Version: "2.0",
				Method:  "eth_subscription",
				Params:  params,
			}
			codec.Write(ctx, item)
		case msgs := <-w.readOperation:
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

func (w *websocketHandler) websocketReadLoop(codec ServerCodec) {
	for {
		select {
		case <-w.close:
			return
		default:
			msgs, _, err := codec.Read()
			if _, ok := err.(*json.SyntaxError); ok {
				codec.Write(context.Background(), errorMessage(err))
			}
			if err != nil {
				w.readErr <- err
				return
			}
			w.readOperation <- msgs
		}
	}
}
