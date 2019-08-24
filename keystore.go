package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/prysmaticlabs/eth1-mock-rpc/eth1"
)

type unencryptedKeysContainer struct {
	Keys []*unencryptedKeys `json:"keys"`
}

type unencryptedKeys struct {
	ValidatorKey  []byte `json:"validator_key"`
	WithdrawalKey []byte `json:"withdrawal_key"`
}

func parseUnencryptedKeysFile(r io.Reader) ([][]byte, [][]byte, error) {
	encoded, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	var ctnr *unencryptedKeysContainer
	if err := json.Unmarshal(encoded, &ctnr); err != nil {
		return nil, nil, err
	}
	validatorKeys := make([][]byte, 0)
	withdrawalKeys := make([][]byte, 0)
	for _, item := range ctnr.Keys {
		validatorKeys = append(validatorKeys, item.ValidatorKey)
		withdrawalKeys = append(withdrawalKeys, item.WithdrawalKey)
	}
	return validatorKeys, withdrawalKeys, nil
}

func createDepositDataFromKeys(validatorKeys [][]byte, withdrawalKeys [][]byte) ([]*eth1.DepositData, error) {
	if len(validatorKeys) != len(withdrawalKeys) {
		return nil, fmt.Errorf("received different number of validator keys %d and withdrawal keys %d", len(validatorKeys), len(withdrawalKeys))
	}
	depositDataItems := make([]*eth1.DepositData, len(validatorKeys))
	for i := 0; i < len(depositDataItems); i++ {
		valSecretKey := validatorKeys[i]
		withdrawalSecretKey := withdrawalKeys[i]
		data, err := eth1.CreateDepositData(valSecretKey, withdrawalSecretKey, eth1.MaxEffectiveBalance)
		if err != nil {
			return nil, err
		}
		depositDataItems[i] = data
	}
	return depositDataItems, nil
}
