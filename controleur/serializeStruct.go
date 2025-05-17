package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"math/big"
	"strconv"
	"time"
)

type SerializablePublicKey struct {
	X     string `json:"x"`     // Coordonnée X encodée en Base64
	Y     string `json:"y"`     // Coordonnée Y encodée en Base64
	Curve string `json:"curve"` // Nom de la courbe (ex: "P-256")
}

func FromECDSAPublicKey(pub *ecdsa.PublicKey) SerializablePublicKey {
	// Encoder les coordonnées X et Y en Base64
	xBytes := pub.X.Bytes()
	yBytes := pub.Y.Bytes()

	spk := SerializablePublicKey{
		X:     base64.StdEncoding.EncodeToString(xBytes),
		Y:     base64.StdEncoding.EncodeToString(yBytes),
		Curve: pub.Curve.Params().Name, // Obtenir le nom de la courbe
	}
	return spk
}

func ToECDSAPublicKey(spk SerializablePublicKey) *ecdsa.PublicKey {
	// Décoder les coordonnées X et Y depuis Base64
	xBytes, _ := base64.StdEncoding.DecodeString(spk.X)
	yBytes, _ := base64.StdEncoding.DecodeString(spk.Y)

	// Convertir les bytes en *big.Int
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	// Trouver la courbe par son nom
	curve := elliptic.P256()

	// Créer la clé publique ecdsa.PublicKey
	pub := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}

	return pub
}

type SerializableTransaction struct {
	Id        int `json:"Id"`
	Sender    SerializablePublicKey
	Receiver  SerializablePublicKey
	Amount    int       `json:"Amount"`
	Timestamp time.Time `json:"Timestamp"`
	Signature []byte    `json:"Signature" hash:"-"`
}

func (transaction *Transaction) FromTransaction() SerializableTransaction {
	return SerializableTransaction{transaction.Id, FromECDSAPublicKey(&transaction.Sender), FromECDSAPublicKey(&transaction.Receiver), transaction.Amount, transaction.Timestamp, transaction.Signature}
}

func (transaction *SerializableTransaction) ToTransaction() Transaction {
	return Transaction{transaction.Id, *ToECDSAPublicKey(transaction.Sender), *ToECDSAPublicKey(transaction.Receiver), transaction.Amount, transaction.Timestamp, transaction.Signature}
}

func printTransactionsId(transactions []Transaction) string {
	var strId string
	for i, tx := range transactions {
		strId += strconv.Itoa(tx.Id) + " "
		if i != len(transactions)-1 {
			strId += "; "
		}
	}
	return strId
}

type SerializableUTXO struct {
	Owner  SerializablePublicKey
	Amount int `json:"Amount"`
}

type SerializableUTXOSet struct {
	Utxos []SerializableUTXO
}

func (utxoSet *UTXOSet) FromUTXOSet() SerializableUTXOSet {
	var serializableUTXOSet SerializableUTXOSet
	for _, utxo := range utxoSet.Utxos {
		tmp := SerializableUTXO{FromECDSAPublicKey(&utxo.Owner), utxo.Amount}
		serializableUTXOSet.Utxos = append(serializableUTXOSet.Utxos, tmp)
	}
	return serializableUTXOSet
}

func (serializable *SerializableUTXOSet) ToUTXOSet() UTXOSet {
	var utxoSet UTXOSet
	for _, utxo := range serializable.Utxos {
		tmp := UTXO{*ToECDSAPublicKey(utxo.Owner), utxo.Amount}
		utxoSet.Utxos = append(utxoSet.Utxos, tmp)
	}
	return utxoSet
}

type SerializableBlock struct {
	Hash         [32]byte                  `json:"hash"`
	PreviousHash [32]byte                  `json:"PreviousHash"`
	Transactions []SerializableTransaction `json:"transactions"`
	UTXOs        SerializableUTXOSet       `json:"UTXOs"`
	Timestamp    time.Time                 `json:"Timestamp"`
	Nonce        uint32                    `json:"Nonce"`
}

func (block *Block) FromBlock() SerializableBlock {
	var serializableTransactions []SerializableTransaction
	for _, tx := range block.Transactions {
		serializableTransactions = append(serializableTransactions, tx.FromTransaction())
	}
	return SerializableBlock{block.Hash, block.PreviousHash, serializableTransactions, block.UTXOs.FromUTXOSet(), block.Timestamp, block.Nonce}
}

func (serializable *SerializableBlock) ToBlock() Block {
	var transactions []Transaction
	for _, tx := range serializable.Transactions {
		transactions = append(transactions, tx.ToTransaction())
	}
	return Block{serializable.Hash, serializable.PreviousHash, transactions, serializable.UTXOs.ToUTXOSet(), serializable.Timestamp, serializable.Nonce}
}

type SerializableBlockchain struct {
	Chain []SerializableBlock `json:"Blockchain"`
}

func (blockchain *Blockchain) FromBlockchain() SerializableBlockchain {
	var serializableBlockchain SerializableBlockchain
	var serializableBlocks []SerializableBlock
	for _, bloc := range blockchain.Chain {
		serializableBlocks = append(serializableBlocks, bloc.FromBlock())
	}
	serializableBlockchain.Chain = serializableBlocks
	return serializableBlockchain
}
