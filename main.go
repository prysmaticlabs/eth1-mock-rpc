package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
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
	keystorePath = flag.String("keystore-path", "", "Path to a validator keystore directory")
	password     = flag.String("password", "", "Password to unlocking the validator keystore directory")
	wsPort       = flag.String("ws-port", "7778", "Port on which to serve websocket listeners")
	httpPort     = flag.String("http-port", "7777", "Port on which to serve http listeners")
	log          = logrus.WithField("prefix", "main")
)

type server struct {
	deposits      []*eth1.DepositData
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

	deposits, err := createDepositDataFromKeystore(*keystorePath, *password)
	if err != nil {
		log.Fatalf("Could not create deposit data from keystore directory: %v", err)
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
		deposits:      deposits,
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

func (s *server) WebsocketHandler() http.Handler {
	return websocket.Server{
		Handler: func(conn *websocket.Conn) {
			codec := newWebsocketCodec(conn)
			defer codec.Close()
			// Listen to read events from the codec and dispatch events or errors accordingly.
			go s.read(codec)
			go s.dispatch(codec)
			<-codec.Closed()
		},
	}
}

func (s *server) dispatch(codec ServerCodec) {
	eth := &eth1.Handler{
		Deposits: s.deposits,
	}
	tick := time.NewTicker(time.Second * 10)
	var latestSubID rpc.ID
	for {
		select {
		case <-s.close:
			return
		case err := <-s.readErr:
			log.WithError(err).Error("Could not read data from request")
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
