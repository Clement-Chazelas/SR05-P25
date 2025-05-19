package main

/*
Ce fichier contient des structures de transition, permettant de convertir en string les structures de la blockchain.
Ces structures sont appelées sérializable. Elles peuvent être transformées automatiquement en string JSON par la fonction json.Marshal()
*/
import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"math/big"
	"strconv"
	"time"
)

// SerializablePublicKey Structure reprenant celle d'une clé publique ecdsa.PublicKey
// Les attributs sont désormais encodés en Base64
type SerializablePublicKey struct {
	X     string `json:"x"`     // Coordonnée X encodée en Base64
	Y     string `json:"y"`     // Coordonnée Y encodée en Base64
	Curve string `json:"curve"` // Nom de la courbe (ex: "P-256")
}

// FromECDSAPublicKey convertit une clé publique ecdsa.PublicKey en SerializablePublicKey
func FromECDSAPublicKey(pub *ecdsa.PublicKey) SerializablePublicKey {
	// Convertir les coordonnées en bytes
	xBytes := pub.X.Bytes()
	yBytes := pub.Y.Bytes()

	spk := SerializablePublicKey{
		X:     base64.StdEncoding.EncodeToString(xBytes), // Encodage en Base64
		Y:     base64.StdEncoding.EncodeToString(yBytes), // Encodage en Base64
		Curve: pub.Curve.Params().Name,                   // Obtenir le nom de la courbe
	}
	return spk
}

// ToECDSAPublicKey convertit une clé SerializablePublicKey en ecdsa.PublicKey
func ToECDSAPublicKey(spk SerializablePublicKey) *ecdsa.PublicKey {
	// Décoder les coordonnées X et Y depuis Base64
	xBytes, _ := base64.StdEncoding.DecodeString(spk.X)
	yBytes, _ := base64.StdEncoding.DecodeString(spk.Y)

	// Convertir les bytes en *big.Int
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	curve := elliptic.P256()

	// Création de la clé publique ecdsa.PublicKey
	pub := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}

	return pub
}

// SerializableTransaction est une structure reprenant celle d'une transaction de la blockchain.
// Les adresses sont remplacées par des variables du type SerializablePublicKey
type SerializableTransaction struct {
	Id        int `json:"Id"`
	Sender    SerializablePublicKey
	Receiver  SerializablePublicKey
	Amount    int       `json:"Amount"`
	Timestamp time.Time `json:"Timestamp"`
	Signature []byte    `json:"Signature" hash:"-"`
}

// FromTransaction convertit une Transaction en SerializableTransaction
func (transaction *Transaction) FromTransaction() SerializableTransaction {
	return SerializableTransaction{transaction.Id,
		FromECDSAPublicKey(&transaction.Sender),   // Conversion de la clé de l'expéditeur en SerializablePublicKey
		FromECDSAPublicKey(&transaction.Receiver), // Conversion de la clé du déstinataire en SerializablePublicKey
		transaction.Amount,
		transaction.Timestamp,
		transaction.Signature}
}

// ToTransaction convertit une SerializableTransaction en Transaction
func (transaction *SerializableTransaction) ToTransaction() Transaction {
	return Transaction{transaction.Id,
		*ToECDSAPublicKey(transaction.Sender),   // Conversion de la SerializablePublicKey en ecdsa.PublicKey
		*ToECDSAPublicKey(transaction.Receiver), // Conversion de la SerializablePublicKey en ecdsa.PublicKey
		transaction.Amount,
		transaction.Timestamp,
		transaction.Signature}
}

// printTransactionsId permet de récupérer un string contenant les id des transactions présentes dans une liste
// afin de les afficher.
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

// SerializableUTXO est un UTXO dont la clé du propriétaire a été convertie en SerializablePublicKey.
type SerializableUTXO struct {
	Owner  SerializablePublicKey
	Amount int `json:"Amount"`
}

// SerializableUTXOSet est un UTXOSet contenant une liste de SerializableUTXO.
type SerializableUTXOSet struct {
	Utxos []SerializableUTXO
}

// FromUTXOSet convertit un UTXOSet en SerializableUTXOSet
func (utxoSet *UTXOSet) FromUTXOSet() SerializableUTXOSet {
	var serializableUTXOSet SerializableUTXOSet
	// Parcourt la liste des UTXO
	for _, utxo := range utxoSet.Utxos {
		// La clé de du propriétaire est convertie en SerializablePublicKey pour générer le SerializableUTXO
		tmp := SerializableUTXO{FromECDSAPublicKey(&utxo.Owner), utxo.Amount}
		// Le SerializableUTXO est ajouté à la liste du SerializableUTXOSet
		serializableUTXOSet.Utxos = append(serializableUTXOSet.Utxos, tmp)
	}
	return serializableUTXOSet
}

// ToUTXOSet convertit un SerializableUTXOSet en UTXOSet.
func (serializable *SerializableUTXOSet) ToUTXOSet() UTXOSet {
	var utxoSet UTXOSet
	// Parcourt la liste des SerializableUTXO
	for _, utxo := range serializable.Utxos {
		// La clé du propriétaire est convertie en ecdsa.PublicKey pour générer l'UTXO
		tmp := UTXO{*ToECDSAPublicKey(utxo.Owner), utxo.Amount}
		// L'UTXO est ajouté à la liste de l'UTXOSet
		utxoSet.Utxos = append(utxoSet.Utxos, tmp)
	}
	return utxoSet
}

// SerializableBlock est une structure reprenant celle d'un block de la blockchain.
// La liste des transactions est une liste de SerializableTransaction.
// La liste des UTXO est une liste de SerializableUTXO.
type SerializableBlock struct {
	Hash         [32]byte                  `json:"hash"`
	PreviousHash [32]byte                  `json:"PreviousHash"`
	Transactions []SerializableTransaction `json:"transactions"`
	UTXOs        SerializableUTXOSet       `json:"UTXOs"`
	Timestamp    time.Time                 `json:"Timestamp"`
	Nonce        uint32                    `json:"Nonce"`
}

// FromBlock convertit un Block en SerializableBlock.
func (block *Block) FromBlock() SerializableBlock {
	var serializableTransactions []SerializableTransaction
	// Parcourt la liste des transactions du block
	for _, tx := range block.Transactions {
		// La transaction est convertie en SerializableTransaction et ajoutée à la liste du nouveau block
		serializableTransactions = append(serializableTransactions, tx.FromTransaction())
	}
	return SerializableBlock{block.Hash,
		block.PreviousHash,
		serializableTransactions,
		block.UTXOs.FromUTXOSet(), // Conversion de l'UTXOSet en SerializableUTXOSet
		block.Timestamp,
		block.Nonce}
}

// ToBlock convertit un SerializableBlock en Block.
func (serializable *SerializableBlock) ToBlock() Block {
	var transactions []Transaction
	// Parcourt la liste des SerializableTransaction du block
	for _, tx := range serializable.Transactions {
		// La SerializableTransaction est convertie en Transaction et ajoutée à la liste du nouveau block
		transactions = append(transactions, tx.ToTransaction())
	}
	return Block{serializable.Hash,
		serializable.PreviousHash,
		transactions,
		serializable.UTXOs.ToUTXOSet(), // Conversion du SerializableUTXOSet en UTXOSet
		serializable.Timestamp,
		serializable.Nonce}
}

// SerializableBlockchain est une structure reprenant celle d'une blockchain.
// La liste des blocks est une liste de SerializableBlock.
type SerializableBlockchain struct {
	Chain []SerializableBlock `json:"Blockchain"`
}

// FromBlockchain convertit une Blockchain en SerializableBlockchain.
func (blockchain *Blockchain) FromBlockchain() SerializableBlockchain {
	var serializableBlocks []SerializableBlock
	// Parcourt la liste des blocks de la blockchain
	for _, bloc := range blockchain.Chain {
		// Le block est converti en SerializableBlock et ajouté à la nouvelle blockchain
		serializableBlocks = append(serializableBlocks, bloc.FromBlock())
	}
	return SerializableBlockchain{serializableBlocks}
}

// ToBlockchain convertit une SerializableBlockchain en Blockchain.
func (serializable *SerializableBlockchain) ToBlockchain() Blockchain {
	var blocks []Block
	// Parcourt la liste des SerializableBlock de la blockchain
	for _, block := range serializable.Chain {
		// Le SerializableBlock est converti en Block et ajouté à la nouvelle blockchain
		blocks = append(blocks, block.ToBlock())
	}
	return Blockchain{blocks}
}
