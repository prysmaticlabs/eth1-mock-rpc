package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"

	"github.com/prysmaticlabs/eth1-mock-rpc/eth1"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"gopkg.in/yaml.v3"
)

const (
	withdrawalPrivkeyFileName = "/shardwithdrawalkey"
	validatorPrivkeyFileName  = "/validatorprivatekey"
)

func createDepositDataFromKeystore(directory string, password string) ([]*eth1.DepositData, error) {
	if directory == "" || password == "" {
		return nil, errors.New("expected a path to the validator keystore and password to be provided, received nil")
	}
	ks := keystore.NewKeystore(directory)
	withdrawalKeys, err := ks.GetKeys(directory, withdrawalPrivkeyFileName, password)
	if err != nil {
		return nil, err
	}
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
	depositDataItems := make([]*eth1.DepositData, len(valMapKeys))
	for i := 0; i < len(depositDataItems); i++ {
		valSecretKey := validatorKeys[valMapKeys[i]].SecretKey.Marshal()
		withdrawalSecretKey := withdrawalKeys[withdrawalMapKeys[i]].SecretKey.Marshal()
		data, err := eth1.CreateDepositData(valSecretKey, withdrawalSecretKey, eth1.MaxEffectiveBalance)
		if err != nil {
			return nil, err
		}
		depositDataItems[i] = data
	}
	return depositDataItems, nil
}

func createValidatorKeysFromKeystore(directory string, password string) ([]string, []string, error) {
	if directory == "" || password == "" {
		return nil, nil, errors.New("expected a path to the validator keystore and password to be provided, received nil")
	}
	ks := keystore.NewKeystore(directory)

	validatorKeys, err := ks.GetKeys(directory, validatorPrivkeyFileName, password)
	if err != nil {
		return nil, nil, err
	}
	valMapKeys := []string{}
	for k := range validatorKeys {
		valMapKeys = append(valMapKeys, k)
	}
	pubKeys := []string{}
	privKeys := []string{}
	for i := 0; i < len(valMapKeys); i++ {
		pk := hex.EncodeToString(validatorKeys[valMapKeys[i]].PublicKey.Marshal())
		sk := hex.EncodeToString(validatorKeys[valMapKeys[i]].SecretKey.Marshal())
		pubKeys = append(pubKeys, pk)
		privKeys = append(privKeys, sk)
	}

	return pubKeys, privKeys, nil
}

func retrieveDepositData(r io.Reader) ([]*eth1.DepositData, error) {
	encodedData, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var deposits []*eth1.DepositData
	if err := json.Unmarshal(encodedData, &deposits); err != nil {
		return nil, err
	}
	return deposits, nil
}

func persistDepositData(w io.Writer, deposits []*eth1.DepositData) error {
	encodedData, err := json.Marshal(deposits)
	if err != nil {
		return err
	}
	if _, err := w.Write(encodedData); err != nil {
		return err
	}
	return nil
}

func persistValidatorDepositData(w io.Writer, pubkeys []string, privkeys []string, deposits []*eth1.DepositData) error {

	type Dummy struct {
		Pubkey                string 
		WithdrawalCredentials string
		Amount                uint64 
		Signature             string
	}
	type DepositDataAndKeys struct {
		DepositData *Dummy
		PubKey      string
		PrivKey     string
	}

	type FML struct {
		DepositDataKeys []*DepositDataAndKeys
	}
	//fmlData := make([]*eth1.DepositData, len(valMapKeys))
	depositDataAndKeys := make([]*DepositDataAndKeys, len(deposits))
	for i := 0; i < len(deposits); i++ {
    
		dummy := &Dummy{Pubkey: hex.EncodeToString(deposits[i].Pubkey), WithdrawalCredentials: hex.EncodeToString(deposits[i].WithdrawalCredentials), Amount: deposits[i].Amount, Signature: hex.EncodeToString(deposits[i].Signature)}
		depositDataAndKeys[i] = &DepositDataAndKeys{ DepositData: dummy, PubKey: pubkeys[i], PrivKey: privkeys[i]}
	}
	fmlData := FML{ DepositDataKeys:depositDataAndKeys}
	yamlOutput, err := yaml.Marshal(fmlData)
	if err != nil {
		return err
	}
	if _, err := w.Write(yamlOutput); err != nil {
		return err
	}
	return nil
}
