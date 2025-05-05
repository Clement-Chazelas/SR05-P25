package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

var blockChain Blockchain

var adressOfSite []ecdsa.PublicKey
var pendingTransactions []Transaction

var sitePrivKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
var sitePubKey = sitePrivKey.PublicKey

var nbSite = 2

var l = log.New(os.Stderr, "", 0)

func sendInitialisation() {
	isKeySent := false
	for {
		if !isKeySent {
			mutex.Lock()
			l.Println(os.Getpid(), sitePubKey)
			fmt.Printf("K:%s\n", SendPublicKey(&sitePubKey))
			mutex.Unlock()
			isKeySent = true
			time.Sleep(time.Duration(1) * time.Second)
		}

		if len(adressOfSite) >= nbSite-1 && sitePubKey.X.Cmp(adressOfSite[0].X) == 1 {
			//j'ai reçu tt les clés + j'ai la clé la plus grande
			l.Println(os.Getpid(), "Je suis l'initialisateur")
			mutex.Lock()
			blockChain.InitBlockchain(append(adressOfSite, sitePubKey))
			fmt.Printf("B:%s\n", SendBlock(blockChain.GetLastBlock()))
			mutex.Unlock()
			l.Println(os.Getpid(), "Initialisation terminée")
			go sendMain()
			break
		} else if len(blockChain.Chain) > 0 {
			// J'attends le premier bloc de la chaine
			l.Println(os.Getpid(), "Initialisation terminée")
			go sendMain()
			break
		}

		time.Sleep(time.Duration(1) * time.Second)
	}
}

func sendMain() {
	for {
		// Remplacer le 2 par autorisation de write
		if len(pendingTransactions) > 0 && sitePubKey.X.Cmp(adressOfSite[0].X) == 1 {
			mutex.Lock()
			l.Println(os.Getpid(), "Initialisation d'un nouveau bloc")
			newBlock := InitBlock(pendingTransactions, blockChain.GetLastBlock().Hash, blockChain.GetLastBlock().UTXOs)
			newBlock.MineBlock()
			blockChain.AddBlock(newBlock)
			fmt.Printf("B:%s\n", SendBlock(blockChain.GetLastBlock()))
			mutex.Unlock()
			pendingTransactions = []Transaction{}
			time.Sleep(time.Duration(5) * time.Second)

		} else if sitePubKey.X.Cmp(adressOfSite[0].X) != 1 {
			mutex.Lock()
			newTransaction := InitTransaction(sitePubKey, adressOfSite[0], 2)
			newTransaction.Sign(sitePrivKey)
			fmt.Printf("T:%s\n", SendTransaction(&newTransaction))
			mutex.Unlock()
			l.Println(os.Getpid(), "Nouvelle transaction envoyée")
			time.Sleep(time.Duration(10) * time.Second)
		}
	}

}

func receive() {
	var rcvmsg string

	for {
		fmt.Scanln(&rcvmsg)
		mutex.Lock()
		if rcvmsg[0:1] == "K" {
			otherKey := ReceivePublicKey(rcvmsg[2:])
			adressOfSite = append(adressOfSite, otherKey)
			l.Println(os.Getpid(), "Clé des autres : ", adressOfSite)

		} else if rcvmsg[0:1] == "T" {
			l.Println(os.Getpid(), "Nouvelle transaction reçu")
			newTransaction := ReceiveTransaction(rcvmsg[2:])
			if newTransaction.Verify(&blockChain) {
				pendingTransactions = append(pendingTransactions, newTransaction)
				l.Println(os.Getpid(), "Nouvelle transaction ajoutée")
			} else {
				l.Println(os.Getpid(), "Nouvelle transaction refusée")
			}

		} else if rcvmsg[0:1] == "B" {
			// vérifier la blockChain
			l.Println(os.Getpid(), "Nouveau bloc reçu")
			newBlock := ReceiveBlock(rcvmsg[2:])
			if len(blockChain.Chain) == 0 {
				//Bloc d'origine, il n'y a pas de vérification
				blockChain.AddBlock(newBlock)

			} else {
				if newBlock.VerifyBlock(blockChain) {
					blockChain.AddBlock(newBlock)
					l.Println(os.Getpid(), "Nouveau bloc ajouté")
					//l.Println(os.Getpid(), blockChain)
				} else {
					l.Println(os.Getpid(), "Nouveau bloc rejeté")
				}
			}
		}
		mutex.Unlock()
		rcvmsg = ""
	}
}

var mutex = &sync.Mutex{}

func main() {

	go sendInitialisation()
	go receive()
	for {
		time.Sleep(time.Duration(60) * time.Second)
	} // Pour attendre la fin des goroutines...*/
}
