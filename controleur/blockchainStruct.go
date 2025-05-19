package main

/*
Ce fichier est une version allégée des structures originales présentes dans le dossier application.
Il n'est utilisé que pour importer les structures et fonctions de l'application nécessaires au controleur.
Toute la documentation des structures et fonctions est disponible dans le fichier du projet application.
*/

import (
	"crypto/ecdsa"
	"encoding/json"
	"time"
)

type Blockchain struct {
	Chain []Block
}
type Block struct {
	Hash         [32]byte
	PreviousHash [32]byte
	Transactions []Transaction
	UTXOs        UTXOSet
	Timestamp    time.Time
	Nonce        uint32
}

type Transaction struct {
	Id        int
	Sender    ecdsa.PublicKey
	Receiver  ecdsa.PublicKey
	Amount    int
	Timestamp time.Time
	Signature []byte `hash:"-"`
}
type UTXO struct {
	Owner  ecdsa.PublicKey
	Amount int
}

type UTXOSet struct {
	Utxos []UTXO
}

func SendBlockchain(blockchain Blockchain) string {
	serializableBlockchain := blockchain.FromBlockchain()
	jsonData, _ := json.Marshal(serializableBlockchain)
	return string(jsonData)
}

func ReceiveBlockchain(data string) SerializableBlockchain {
	var serializable SerializableBlockchain
	err := json.Unmarshal([]byte(data), &serializable)

	if err != nil {
		return SerializableBlockchain{}
	}
	return serializable
}
