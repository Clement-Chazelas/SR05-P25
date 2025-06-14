package main

/*
Fichier contenant les structures et les fonctions nécessaires à la création d'une blockchain.
Pour des raisons de simplification, les adresses sont équivalentes aux clés publiques.
*/

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

// Blockchain Structure contenant une liste de blocks formant une blockchain
type Blockchain struct {
	Chain []Block
}

// AddBlock Fonction permettant d'ajouter un nouveau block à la fin d'une blockchain
func (blockchain *Blockchain) AddBlock(newBlock Block) {
	blockchain.Chain = append(blockchain.Chain, newBlock)
}

// GetLastBlock Fonction retournant le dernier block d'une blockchain
func (blockchain *Blockchain) GetLastBlock() *Block {
	return &blockchain.Chain[len(blockchain.Chain)-1]
}

// InitBlockchain Fonction d'initialisation d'une blockchain.
// Création du premier block ajoutant 1000 coins à la liste d'adresses fournie.
func (blockchain *Blockchain) InitBlockchain(keys []ecdsa.PublicKey) {
	var firstBlock Block
	var utxoSet UTXOSet // Liste des montants possédés par chaque adresse
	for _, key := range keys {
		// Ajout de 1000 coins à chaque adresse fournie dans la liste
		utxoSet.Utxos = append(utxoSet.Utxos, UTXO{key, 1000})
	}
	firstBlock.UTXOs = utxoSet
	firstBlock.Timestamp = time.Now()
	firstBlock.MineBlock()
	blockchain.AddBlock(firstBlock)
}

// Block Structure contenant les informations d'un block.
type Block struct {
	Hash         [32]byte      // Hash du block
	PreviousHash [32]byte      // Hash du block précédent
	Transactions []Transaction // Liste des transactions du block
	UTXOs        UTXOSet       // Liste des UTXO mise à jour en fonction des transactions du block
	Timestamp    time.Time     // Timestamp du block
	Nonce        uint32        // Valeur permettant de remplir les conditions sur le hash du block
}

// InitBlock Fonction d'initialisation d'un bloc avant son minage.
// Renvoie un nouveau block contenant la liste des transactions fournies et met à jour les UTXO en fonction de ces dernières.
func InitBlock(transac []Transaction, prevHash [32]byte, prevUTXOs UTXOSet) Block {
	var newBlock Block

	newBlock.PreviousHash = prevHash
	newBlock.Transactions = transac
	//Calcule des nouveaux UTXO
	newBlock.UTXOs = CalculateUTXOs(prevUTXOs, transac)
	newBlock.Timestamp = time.Now()

	return newBlock
}

// Concatenate Fonction permettant de concaténer les différents attributs d'un block pour le calcul du hash.
func (block *Block) Concatenate() []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	encoder.Encode(block.PreviousHash)
	encoder.Encode(block.Transactions)
	encoder.Encode(block.Timestamp)
	encoder.Encode(block.UTXOs)

	concat := buffer.Bytes()
	// Si le block a déjà été miné, on ajoute le nonce à la concaténation
	if block.Nonce != 0 {
		tmp := make([]byte, 4)
		// Conversion du nonce en bytes
		binary.LittleEndian.PutUint32(tmp, block.Nonce)
		concat = append(concat, tmp...)
	}
	return concat
}

// MineBlock Fonction permettant de miner un block.
// Recherche du Nonce permettant de remplir les conditions sur le hash du block.
func (block *Block) MineBlock() {
	// Récupération des attributs du block concaténés pour le calcul du hash
	concat := block.Concatenate()
	// Heure utilisée pour mesurer la durée de minage
	heureDebut := time.Now()

	for tempNonce := uint32(1); tempNonce < 1000000000; tempNonce++ {
		// Conversion du nonce en bytes
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, tempNonce)
		// Ajout du nonce à la fin des attributs concaténés
		tmp = append(concat, tmp...)
		// Calcul du Hash du block
		hash := sha256.Sum256(tmp)

		strHash := fmt.Sprintf("%x", hash)
		// Vérification que le hash respecte les conditions (commence par 00000)
		if strHash[0:5] == "00000" {

			block.Nonce = tempNonce
			block.Hash = hash
			// Le block a été correctement miné
			stderr.Printf(" %s%s Bloc miné en : %s %s\n", magenta, Nom, time.Since(heureDebut), raz)
			break
		}

	}
}

// VerifyBlock Fonction permettant de vérifier qu'un block respecte les conditions de la blockchain
// et qu'il n'a pas été altéré.
func (block *Block) VerifyBlock(chain Blockchain) bool {
	concat := block.Concatenate()
	// Calcul du hash du block
	hash := sha256.Sum256(concat)
	strHash := fmt.Sprintf("%x", hash)
	// Récupération du block précédent
	lastBlock := chain.GetLastBlock()

	// Vérification du hash du block
	if hash != block.Hash || strHash[0:5] != "00000" || lastBlock.Hash != block.PreviousHash {
		// Si le hash calculé est différent du hash stocké dans le block
		// Si le hash ne commence pas par 00000
		// Si le hash du block précédent n'est pas le bon
		return false
	}

	// Vérification des transactions
	for _, tx := range block.Transactions {
		// Chaque transaction présente dans le block est vérifiée
		if !tx.Verify(&chain) {
			stderr.Println("Transaction verification failed")
			return false
		}
	}

	// Vérifier l'UTXOSet : l'UTXO (solde) de chaque adresse est recalculé en fonction des transactions
	// et comparé à celui stocké dans le block
	for _, ut := range CalculateUTXOs(lastBlock.UTXOs, block.Transactions).Utxos {
		if ut.Amount != block.UTXOs.FindByKey(ut.Owner).Amount {
			// Le montant calculé est différent du montant stocké dans le block
			return false
		}
	}

	return true
}

// updateTransactionsFromBlock Fonction permettant de mettre à jour la liste des transactions en attente
// en fonction d'un nouveau block reçu.
// Les transactions présentent dans le blocks sont retirées de la nouvelle liste d'attente.
func (block *Block) updateTransactionsFromBlock(transactions []Transaction) []Transaction {
	var newTransactions []Transaction
	// Parcourt de la liste des transactions en attente
	for _, tx := range transactions {
		isInside := false
		// Parcourt de la liste des transactions du block
		for _, txBlock := range block.Transactions {
			// Si la transaction est présente dans le block, elle n'a pas ajouté à la nouvelle liste
			if bytes.Equal(tx.Signature, txBlock.Signature) {
				isInside = true
				break
			}
		}
		// Le block a été parcouru et la transaction n'a pas été trouvée
		if !isInside {
			// Elle est ajoutée à la nouvelle liste
			newTransactions = append(newTransactions, tx)
		}
	}
	return newTransactions

}

// Transaction Structure contenant les informations d'une transaction de la blockchain.
type Transaction struct {
	Id        int             // Identifiant de la transaction (généré aléatoirement, aide visuelle)
	Sender    ecdsa.PublicKey // Adresse (clé publique) de l'expéditeur
	Receiver  ecdsa.PublicKey // Adresse du destinataire
	Amount    int             // Montant de la transaction
	Timestamp time.Time       // Timestamp de la transaction
	Signature []byte          `hash:"-"` // Signature du hash des autres attributs de la transaction
}

// InitTransaction Fonction permettant d'initialiser une transaction.
// Prend en paramètre l'adresse de l'expéditeur, l'adresse du destinataire et le montant de la transaction.
func InitTransaction(sender ecdsa.PublicKey, receiver ecdsa.PublicKey, amount int) Transaction {
	var transaction Transaction
	transaction.Id = mrand.Intn(9999) + 1 // Id généré aléatoirement
	transaction.Sender = sender
	transaction.Receiver = receiver
	transaction.Amount = amount
	// Le timestamp est préparé pour pouvoir être print
	jsonData, _ := json.Marshal(time.Now())
	json.Unmarshal(jsonData, &transaction.Timestamp)

	return transaction
}

// Sign Fonction permettant de signer une transaction.
// Le hash des attributs concaténés de la transaction est signé avec la clé privée de l'expéditeur.
func (transaction *Transaction) Sign(privKey *ecdsa.PrivateKey) {
	// Hashage des attributs
	hash := structhash.Sha256(transaction, 1)
	// Signature du hash utilisant les courbes elliptiques
	sig, _ := ecdsa.SignASN1(rand.Reader, privKey, hash[:])
	transaction.Signature = sig
}

// Verify Fonction permettant de vérifier qu'une transaction est valide.
// Vérification de la signature et que l'expéditeur possède bien l'argent nécessaire.
func (transaction *Transaction) Verify(chain *Blockchain) bool {

	// Verification de la Signature
	// Recalcule du hash des attributs de la transaction
	hash := structhash.Sha256(transaction, 1)

	// Vérification que le hash calculé correspond à la signature déchiffrée
	// à l'aide de la clé publique de l'expéditeur
	if !ecdsa.VerifyASN1(&transaction.Sender, hash[:], transaction.Signature) {
		stderr.Println("Signature verification failed")
		return false
	}

	// Récupération des UTXO de l'expéditeur
	senderUTXO := chain.GetLastBlock().UTXOs.FindByKey(transaction.Sender)
	// Vérification que l'expéditeur possède les coins suffisants
	if senderUTXO.Amount < transaction.Amount {
		stderr.Println("Not enough UTXO")
		return false
	}
	return true
}

// UTXO Structure contenant le montant de coins possédés par une adresse.
type UTXO struct {
	Owner  ecdsa.PublicKey // Adresse du propriétaire (clé publique)
	Amount int             // Montant des coins possédés
}

// UTXOSet Structure contenant la liste des UTXO de chaque adresse de la blockchain.
type UTXOSet struct {
	Utxos []UTXO
}

// FindByKey Fonction renvoyant l'UTXO correspondant à une adresse dans l'UTXOSet.
func (utxoSet *UTXOSet) FindByKey(owner ecdsa.PublicKey) *UTXO {
	for i, v := range utxoSet.Utxos {
		if AreKeyEquals(v.Owner, owner) {
			return &utxoSet.Utxos[i]
		}
	}
	return nil
}

// AreKeyEquals Vérifie que deux adresses (clés publiques) sont égales.
func AreKeyEquals(a, b ecdsa.PublicKey) bool {
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0 && a.Curve == b.Curve
}

// CalculateUTXOs Fonction calculant le nouvel UTXOSet en fonction d'une liste de transactions
func CalculateUTXOs(utxos UTXOSet, transactions []Transaction) UTXOSet {
	var newUTXOs UTXOSet
	// Copie de l'ancien UTXOSet pour éviter qu'il soit modifié
	newUTXOs.Utxos = make([]UTXO, len(utxos.Utxos))
	for i := range utxos.Utxos {
		newUTXOs.Utxos[i] = utxos.Utxos[i]
	}
	// Parcourt de la liste des transactions
	for _, tx := range transactions {
		// Si le receveur n'existe pas encore (nouveau site), on l'ajoute
		if newUTXOs.FindByKey(tx.Receiver) == nil {
			newUTXOs.Utxos = append(newUTXOs.Utxos, UTXO{tx.Receiver, 0})
		}
		// Le montant de la transaction est enlevé à l'expéditeur et ajouté au destinataire
		newUTXOs.FindByKey(tx.Sender).Amount -= tx.Amount
		newUTXOs.FindByKey(tx.Receiver).Amount += tx.Amount
	}

	return newUTXOs
}

// isTheBiggestKey Fonction vérifiant si une clé publique est la plus grande de la liste fournie.
func isTheBiggestKey(key1 ecdsa.PublicKey, keys []ecdsa.PublicKey) bool {
	// Parcourt de la liste des clés fournie
	for _, key := range keys {
		// Si la clé est inférieure, retourne false
		if key1.X.Cmp(key.X) == -1 {
			return false
		}
	}
	return true
}

// SendPublicKey Fonction permettant de convertir une clé en string, permettant de la print.
func SendPublicKey(key *ecdsa.PublicKey) string {
	// Conversion de la clé en clé sérialisable
	serializableKey := FromECDSAPublicKey(key)
	// Conversion de la clé sérialisable en string JSON
	jsonData, _ := json.Marshal(serializableKey)
	return string(jsonData)
}

// ReceivePublicKey Fonction permettant récupérer une clé à partir d'un string.
func ReceivePublicKey(data string) ecdsa.PublicKey {
	var Serializable SerializablePublicKey
	// Récupération de la clé sérialisable à partir du string
	err := json.Unmarshal([]byte(data), &Serializable)
	if err != nil {
		return ecdsa.PublicKey{}
	}
	// Conversion de la clé sérialisable en clé
	return *ToECDSAPublicKey(Serializable)
}

// SendTransaction Fonction permettant de convertir une transaction en string, permettant de la print.
func SendTransaction(transaction *Transaction) string {
	// Conversion de la transaction en transaction sérialisable
	serializableTransac := transaction.FromTransaction()
	// Conversion de la transaction sérialisable en string JSON
	jsonData, _ := json.Marshal(serializableTransac)
	return string(jsonData)
}

// ReceiveTransaction Fonction permettant de récupérer une transaction à partir d'un string.
func ReceiveTransaction(data string) Transaction {
	var Serializable SerializableTransaction
	// Récupération de la transaction sérialisable à partir du string
	err := json.Unmarshal([]byte(data), &Serializable)
	if err != nil {
		return Transaction{}
	}
	// Conversion de la transaction sérialisable en transaction
	return Serializable.ToTransaction()
}

// SendBlock Fonction permettant de convertir une block en string, permettant de le print.
func SendBlock(block *Block) string {
	// Conversion du block en block sérialisable
	serializeBlock := block.FromBlock()
	// Conversion du block sérialisable en string JSON
	jsonData, _ := json.Marshal(serializeBlock)
	return string(jsonData)
}

// ReceiveBlock Fonction permettant de récupérer un block à partir d'un string.
func ReceiveBlock(data string) Block {
	var Serializable SerializableBlock
	// Récupération du block sérialisable à partir du string
	err := json.Unmarshal([]byte(data), &Serializable)
	if err != nil {
		return Block{}
	}
	// Conversion du block sérialisable en block
	return Serializable.ToBlock()
}

// SendBlockchain Fonction permettant de convertir une blockchain en string, permettant de la print.
func SendBlockchain(blockchain Blockchain) string {
	// Conversion de la blockchain en blockchain sérialisable
	serializableBlockchain := blockchain.FromBlockchain()
	// Conversion de la blockchain sérialisable en string JSON
	jsonData, _ := json.Marshal(serializableBlockchain)
	return string(jsonData)
}

// ReceiveBlockchain Fonction permettant de récupérer une blockchain à partir d'un string.
func ReceiveBlockchain(data string) SerializableBlockchain {
	var Serializable SerializableBlockchain
	// Récupération de la blockchain sérialisable à partir du string
	err := json.Unmarshal([]byte(data), &Serializable)
	if err != nil {
		return SerializableBlockchain{}
	}
	// Conversion de la blockchain sérialisable en blockchain
	return Serializable
}
