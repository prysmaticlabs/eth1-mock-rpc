package main

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/eth1-mock-rpc/eth1"
)

type mockWriter struct {
	encoded []byte
}

func (w *mockWriter) Write(b []byte) (int, error) {
	w.encoded = b
	return len(w.encoded), nil
}

func TestRoundTripRetrieveDepositData(t *testing.T) {
	deposits := []*eth1.DepositData{
		{
			Pubkey:                []byte{1, 2, 3, 4, 5},
			WithdrawalCredentials: []byte{6, 7, 8, 9, 10},
			Amount:                32,
			Signature:             make([]byte, 96),
		},
		{
			Pubkey:                []byte{9, 9, 9, 9},
			WithdrawalCredentials: []byte{8, 8, 8, 8},
			Amount:                40,
			Signature:             make([]byte, 96),
		},
	}
	w := &mockWriter{}
	if err := persistDepositData(w, deposits); err != nil {
		t.Fatal(err)
	}
	buf := bytes.NewBuffer(w.encoded)
	parsedDeposits, err := retrieveDepositData(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(deposits, parsedDeposits) {
		t.Errorf("Original deposits = %v, received = %v", deposits, parsedDeposits)
	}
}
