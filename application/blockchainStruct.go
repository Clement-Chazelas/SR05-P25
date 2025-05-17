package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/cnf/structhash"
	mrand "math/rand"
	"time"
)

type Blockchain struct {
	Chain []Block
}

// ce qui est () indique que la fonction s'applique sur ce type
func (blockchain *Blockchain) AddBlock(newBlock Block) {
	blockchain.Chain = append(blockchain.Chain, newBlock)
}

func (blockchain *Blockchain) GetLastBlock() *Block {
	return &blockchain.Chain[len(blockchain.Chain)-1]
}

func (blockchain *Blockchain) InitBlockchain(keys []ecdsa.PublicKey) {
	var firstBlock Block
	var utxoSet UTXOSet
	for _, key := range keys {
		utxoSet.Utxos = append(utxoSet.Utxos, UTXO{key, 1000})
	}
	firstBlock.UTXOs = utxoSet
	firstBlock.Timestamp = time.Now()
	firstBlock.MineBlock()
	blockchain.AddBlock(firstBlock)
}

type Block struct {
	Hash         [32]byte
	PreviousHash [32]byte
	Transactions []Transaction
	UTXOs        UTXOSet
	Timestamp    time.Time
	Nonce        uint32 //version 1 / version 2
}

func InitBlock(transac []Transaction, prevHash [32]byte, prevUTXOs UTXOSet) Block {
	var newBlock Block

	newBlock.PreviousHash = prevHash
	newBlock.Transactions = transac
	//Calculer nouveau UTXOs
	newBlock.UTXOs = CalculateUTXOs(prevUTXOs, transac)
	newBlock.Timestamp = time.Now()

	return newBlock
}

func (block *Block) Concatenate() []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	encoder.Encode(block.PreviousHash)
	encoder.Encode(block.Transactions)
	encoder.Encode(block.Timestamp)
	encoder.Encode(block.UTXOs)

	concat := buffer.Bytes()

	if block.Nonce != 0 {
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, block.Nonce)
		concat = append(concat, tmp...)
	}
	return concat
}

func (block *Block) MineBlock() {

	concat := block.Concatenate()
	heureDebut := time.Now()

	for i := uint32(1); i < 1000000000; i++ {

		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, i)

		tmp = append(concat, tmp...)

		hash := sha256.Sum256(tmp)

		strHash := fmt.Sprintf("%x", hash)

		if strHash[0:5] == "00000" {

			block.Nonce = i
			block.Hash = hash

			stderr.Printf(" %s%s Bloc miné en : %s %s\n", magenta, Nom, time.Since(heureDebut), raz)
			break
		}

	}
}

func (block *Block) VerifyBlock(chain Blockchain) bool {
	concat := block.Concatenate()

	hash := sha256.Sum256(concat)
	strHash := fmt.Sprintf("%x", hash)
	lastBlock := chain.GetLastBlock()
	// Vérification du hash
	if hash != block.Hash || strHash[0:5] != "00000" || lastBlock.Hash != block.PreviousHash {
		return false
	}

	// Vérification des transactions
	for _, tx := range block.Transactions {
		if !tx.Verify(&chain) {
			stderr.Println("Transaction verification failed")
			return false
		}
	}

	// Vérifier UTXOSet
	for _, ut := range CalculateUTXOs(lastBlock.UTXOs, block.Transactions).Utxos {
		if ut.Amount != block.UTXOs.FindByKey(ut.Owner).Amount {
			return false
		}
	}

	return true
}

func (block *Block) updateTransactionsFromBlock(transactions []Transaction) []Transaction {
	var newTransactions []Transaction
	for _, tx := range transactions {
		isInside := false
		for _, txBlock := range block.Transactions {
			if bytes.Equal(tx.Signature, txBlock.Signature) {
				isInside = true
				break
			}
		}
		if !isInside {
			newTransactions = append(newTransactions, tx)
		}
	}
	return newTransactions

}

type Transaction struct {
	Id        int
	Sender    ecdsa.PublicKey
	Receiver  ecdsa.PublicKey
	Amount    int
	Timestamp time.Time
	Signature []byte `hash:"-"`
}

/*func (transaction *Transaction) Concatenate() []byte {
	var buffer bytes.Buffer
	byt := structhash.Dump(*transaction, 1)
	encoder := gob.NewEncoder(&buffer)
	encoder.Encode(transaction.Sender)
	encoder.Encode(transaction.Receiver)
	encoder.Encode(transaction.Amount)
	encoder.Encode(transaction.Timestamp)

	l.Println(os.Getpid(), byt[len(byt)-10:len(byt)-1])
	return buffer.Bytes()
}*/

func InitTransaction(sender ecdsa.PublicKey, receiver ecdsa.PublicKey, amount int) Transaction {
	var transaction Transaction
	transaction.Id = mrand.Intn(9999) + 1
	transaction.Sender = sender
	transaction.Receiver = receiver
	transaction.Amount = amount
	jsonData, _ := json.Marshal(time.Now())
	json.Unmarshal(jsonData, &transaction.Timestamp)

	return transaction
}

func (transaction *Transaction) Sign(privKey *ecdsa.PrivateKey) {
	//hash := sha256.Sum256(transaction.Concatenate())
	hash := structhash.Sha256(transaction, 1)
	sig, _ := ecdsa.SignASN1(rand.Reader, privKey, hash[:])
	transaction.Signature = sig
}

func (transaction *Transaction) Verify(chain *Blockchain) bool {

	//Verif de la Signature
	//concat := transaction.Concatenate()
	hash := structhash.Sha256(transaction, 1)
	//hash := sha256.Sum256(concat)
	if !ecdsa.VerifyASN1(&transaction.Sender, hash[:], transaction.Signature) {
		stderr.Println("Signature verification failed")
		return false
	}

	//Verif que le send à l'argent
	senderUTXO := chain.GetLastBlock().UTXOs.FindByKey(transaction.Sender)
	if senderUTXO.Amount < transaction.Amount {
		stderr.Println("Not enough UTXO")
		return false
	}
	return true
}

type UTXO struct {
	Owner  ecdsa.PublicKey
	Amount int
}

type UTXOSet struct {
	Utxos []UTXO
}

func (utxoSet *UTXOSet) FindByKey(owner ecdsa.PublicKey) *UTXO {
	for i, v := range utxoSet.Utxos {
		if AreKeyEquals(v.Owner, owner) {
			return &utxoSet.Utxos[i]
		}
	}
	return nil
}

func AreKeyEquals(a, b ecdsa.PublicKey) bool {
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0 && a.Curve == b.Curve
}

func CalculateUTXOs(utxos UTXOSet, transactions []Transaction) UTXOSet {
	var newUTXOs UTXOSet
	newUTXOs.Utxos = make([]UTXO, len(utxos.Utxos))

	for i := range utxos.Utxos {
		newUTXOs.Utxos[i] = utxos.Utxos[i]
	}

	for _, tx := range transactions {
		newUTXOs.FindByKey(tx.Sender).Amount -= tx.Amount
		newUTXOs.FindByKey(tx.Receiver).Amount += tx.Amount
	}
	//fmt.Println(newUTXOs)
	return newUTXOs
}

func isTheBiggestKey(key1 ecdsa.PublicKey, keys []ecdsa.PublicKey) bool {
	for _, key := range keys {
		if key1.X.Cmp(key.X) == -1 {
			return false
		}
	}
	return true
}

func SendPublicKey(key *ecdsa.PublicKey) string {
	serializableKey := FromECDSAPublicKey(key)
	jsonData, _ := json.Marshal(serializableKey)
	return string(jsonData)
}

func ReceivePublicKey(data string) ecdsa.PublicKey {
	var Serializable SerializablePublicKey
	err := json.Unmarshal([]byte(data), &Serializable)
	if err != nil {
		return ecdsa.PublicKey{}
	}
	return *ToECDSAPublicKey(Serializable)
}

func SendTransaction(transaction *Transaction) string {
	serializableTransac := transaction.FromTransaction()
	jsonData, _ := json.Marshal(serializableTransac)
	return string(jsonData)
}

func ReceiveTransaction(data string) Transaction {
	var Serializable SerializableTransaction
	err := json.Unmarshal([]byte(data), &Serializable)
	if err != nil {
		return Transaction{}
	}
	return Serializable.ToTransaction()
}

func SendBlock(block *Block) string {
	serializeBlock := block.FromBlock()
	jsonData, _ := json.Marshal(serializeBlock)
	return string(jsonData)
}

func ReceiveBlock(data string) Block {
	var Serializable SerializableBlock
	err := json.Unmarshal([]byte(data), &Serializable)
	if err != nil {
		return Block{}
	}
	return Serializable.ToBlock()
}

func SendBlockchain(blockchain Blockchain) string {
	serializableBlockchain := blockchain.FromBlockchain()
	jsonData, _ := json.Marshal(serializableBlockchain)
	return string(jsonData)
}

func ReceiveBlockchain(data string) SerializableBlockchain {
	var Serializable SerializableBlockchain
	err := json.Unmarshal([]byte(data), &Serializable)
	if err != nil {
		return SerializableBlockchain{}
	}
	return Serializable
}

/*func main() {
	//Attention au \n dans les sends
	blockchain := Blockchain{}

	U1PrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	U1PubKey := U1PrivKey.PublicKey

	U2PrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	U2PubKey := U2PrivKey.PublicKey

	strKey := SendPublicKey(&U1PubKey)
	U1Copy := ReceivePublicKey(strKey)

	var block Block
	var utxos UTXOSet

	var utxo1 UTXO

	utxo1.Owner = U1PubKey
	utxo1.Amount = 10

	var utxo2 UTXO

	utxo2.Owner = U2PubKey
	utxo2.Amount = 10

	utxos.Utxos = append(utxos.Utxos, utxo1)
	utxos.Utxos = append(utxos.Utxos, utxo2)

	block.UTXOs = utxos
	block.Timestamp = time.Now()
	block.MineBlock()

	blockchain.AddBlock(block)

	transac := InitTransaction(U1PubKey, U2PubKey, 10)
	transac.Sign(U1PrivKey)

	concat := transac.Concatenate()
	hash := sha256.Sum256(concat)
	if ecdsa.VerifyASN1(&U1Copy, hash[:], transac.Signature) {
		fmt.Println("réussi")
	} else {
		fmt.Println("raté")
	}

	a := blockchain.GetLastBlock().UTXOs.FindByKey(U1Copy)
	if a != nil {
		fmt.Println("réussi")
	} else {
		fmt.Println("raté")
	}

	transSTR := SendTransaction(&transac)
	fmt.Println(transSTR)
	newTrans := ReceiveTransaction(transSTR)

	if transac.Verify(&blockchain) {
		fmt.Println("Transaction verified")
	} else {
		fmt.Println("Transaction not verified")
	}

	var transactions []Transaction

	transactions = append(transactions, transac)
	newblock := InitBlock(transactions, blockchain.GetLastBlock().Hash, blockchain.GetLastBlock().UTXOs)

	newblock.MineBlock()

	blockStr := SendBlock(&newblock)
	newblockCopy := ReceiveBlock(blockStr)

	fmt.Println(blockStr)
	//fmt.Println(newblock.UTXOs.FindByKey(U2PubKey).Amount)
	if newblockCopy.VerifyBlock(blockchain) {
		fmt.Println("New block verified")
	} else {
		fmt.Println("New block not verified")
	}

	blockchain.AddBlock(newblock)

}*/
