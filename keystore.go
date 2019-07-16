package main

import (
	"errors"
	"path"

	"github.com/prysmaticlabs/prysm/shared/keystore"
)

const (
	withdrawalPrivkeyFileName = "/shardwithdrawalkey"
	validatorPrivkeyFileName  = "/validatorprivatekey"
)

func createDepositDataFromKeystore(directory string, password string) ([]*depositData, error) {
	if directory == "" || password == "" {
		return nil, errors.New("expected a path to the validator keystore and password to be provided, received nil")
	}
	log.Infof("Parsing shard withdrawal private keys from %s, this may take a while...", path.Join(directory, withdrawalPrivkeyFileName))
	ks := keystore.NewKeystore(directory)
	withdrawalKeys, err := ks.GetKeys(directory, withdrawalPrivkeyFileName, password)
	if err != nil {
		return nil, err
	}
	log.Infof("Parsing validator private keys from %s, this may take a while...", path.Join(directory, validatorPrivkeyFileName))
	validatorKeys, err := ks.GetKeys(directory, validatorPrivkeyFileName, password)
	if err != nil {
		return nil, err
	}
	if len(validatorKeys) != len(withdrawalKeys) {
		return nil, errors.New("unequal number of validator and withdrawal keys")
	}
	valMapKeys := []string{}
	withdrawalMapKeys := []string{}
	for k := range validatorKeys {
		valMapKeys = append(valMapKeys, k)
	}
	for k := range withdrawalKeys {
		withdrawalMapKeys = append(withdrawalMapKeys, k)
	}
	depositDataItems := make([]*depositData, len(valMapKeys))
	for i := 0; i < len(depositDataItems); i++ {
		valSecretKey := validatorKeys[valMapKeys[i]].SecretKey.Marshal()
		withdrawalSecretKey := withdrawalKeys[withdrawalMapKeys[i]].SecretKey.Marshal()
		data, err := createDepositData(valSecretKey, withdrawalSecretKey, maxEffectiveBalance)
		if err != nil {
			return nil, err
		}
		depositDataItems[i] = data
	}
	return depositDataItems, nil
}
