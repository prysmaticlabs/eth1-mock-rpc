// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"encoding/json"

	"golang.org/x/net/websocket"
)

// websocketJSONCodec is a custom JSON codec with payload size enforcement and
// special number parsing.
var websocketJSONCodec = websocket.Codec{
	// Marshal is the stock JSON marshaller used by the websocket library too.
	Marshal: func(v interface{}) ([]byte, byte, error) {
		msg, err := json.Marshal(v)
		return msg, websocket.TextFrame, err
	},
	// Unmarshal is a specialized unmarshaller to properly convert numbers.
	Unmarshal: func(msg []byte, payloadType byte, v interface{}) error {
		dec := json.NewDecoder(bytes.NewReader(msg))
		dec.UseNumber()

		return dec.Decode(v)
	},
}

// Create a custom encode/decode pair to enforce payload size and number encoding.
func newWebsocketCodec(conn *websocket.Conn) ServerCodec {
	conn.MaxPayloadBytes = maxRequestContentLength
	encoder := func(v interface{}) error {
		return websocketJSONCodec.Send(conn, v)
	}
	decoder := func(v interface{}) error {
		return websocketJSONCodec.Receive(conn, v)
	}
	rpcconn := Conn(conn)
	if conn.IsServerConn() {
		// Override remote address with the actual socket address because
		// package websocket crashes if there is no request origin.
		addr := conn.Request().RemoteAddr
		if wsaddr := conn.RemoteAddr().(*websocket.Addr); wsaddr.URL != nil {
			// Add origin if present.
			addr += "(" + wsaddr.URL.String() + ")"
		}
		rpcconn = connWithRemoteAddr{conn, addr}
	}
	return NewCodec(rpcconn, encoder, decoder)
}
