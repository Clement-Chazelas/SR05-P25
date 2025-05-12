package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

var blockChain Blockchain

var adressOfSite []ecdsa.PublicKey
var pendingTransactions []Transaction

var sitePrivKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
var sitePubKey = sitePrivKey.PublicKey

var nbSite = 3
var allowSC = false

var pNom = flag.String("n", "app", "Nom")
var Nom = *pNom + "-" + strconv.Itoa(os.Getpid())

var l = log.New(os.Stderr, "", 0)

func sendInitialisation() {
	isKeySent := false
	for {
		if !isKeySent {
			//J'envoie ma clé aux autres sites (une seule fois)
			mutex.Lock()

			l.Println(Nom, sitePubKey)
			fmt.Printf("K:%s\n", SendPublicKey(&sitePubKey))

			mutex.Unlock()
			isKeySent = true
			time.Sleep(time.Duration(1) * time.Second)
		}

		if len(adressOfSite) >= nbSite-1 && sitePubKey.X.Cmp(adressOfSite[0].X) == 1 {
			//j'ai reçu tt les clés + j'ai la clé la plus grande
			l.Println(Nom, "Je suis l'initialisateur")
			mutex.Lock()

			blockChain.InitBlockchain(append(adressOfSite, sitePubKey))
			fmt.Printf("B:%s\n", SendBlock(blockChain.GetLastBlock()))

			mutex.Unlock()
			l.Println(Nom, "Initialisation terminée")
			go sendMain()
			break
		} else if len(blockChain.Chain) > 0 {
			// J'attends le premier bloc de la chaine
			l.Println(Nom, "Initialisation terminée")
			go sendMain()
			break
		}

		time.Sleep(time.Duration(1) * time.Second)
	}
}

func sendMain() {
	isSCAsked := false
	for {
		// Si la liste des transactions non vide -> minage
		if len(pendingTransactions) > 0 {
			// Verif si autorisation d'accès à la section critique
			if allowSC {
				mutex.Lock()
				l.Println(Nom, "Initialisation d'un nouveau bloc")

				newBlock := InitBlock(pendingTransactions, blockChain.GetLastBlock().Hash, blockChain.GetLastBlock().UTXOs)
				newBlock.MineBlock()
				blockChain.AddBlock(newBlock)
				fmt.Printf("B:%s\n", SendBlock(blockChain.GetLastBlock()))

				allowSC = false
				isSCAsked = false
				fmt.Printf("FILE:finSC\n")

				mutex.Unlock()
				pendingTransactions = []Transaction{}

				time.Sleep(time.Duration(5) * time.Second)
			} else if !isSCAsked {
				// On ne demande qu'une seule fois l'accès à la section critique
				fmt.Printf("FILE:demandeSC\n")
				isSCAsked = true

				time.Sleep(time.Duration(2) * time.Second)
			} else {
				// J'attends mon autorisation d'accès à la section critique, je continue d'envoyer des transac en attendant
				mutex.Lock()

				newTransaction := InitTransaction(sitePubKey, adressOfSite[0], 2)
				newTransaction.Sign(sitePrivKey)
				fmt.Printf("T:%s\n", SendTransaction(&newTransaction))

				mutex.Unlock()

				l.Println(Nom, "Nouvelle transaction envoyée")

				time.Sleep(time.Duration(5) * time.Second)
			}

		} else {
			//Je n'ai rien à miner donc création et envoi d'une transaction
			mutex.Lock()

			newTransaction := InitTransaction(sitePubKey, adressOfSite[0], 2)
			newTransaction.Sign(sitePrivKey)
			fmt.Printf("T:%s\n", SendTransaction(&newTransaction))

			mutex.Unlock()

			l.Println(Nom, "Nouvelle transaction envoyée")
			time.Sleep(time.Duration(10) * time.Second)
		}
	}

}

func receive() {
	var rcvmsg string

	for {
		fmt.Scanln(&rcvmsg)
		mutex.Lock()

		if rcvmsg[:5] != "CONT:" {
			//Ce message n'était pas à destination de l'application
			mutex.Unlock()
			rcvmsg = ""
			continue
		}

		rcvmsg = rcvmsg[5:]

		if rcvmsg[:2] == "K:" {
			otherKey := ReceivePublicKey(rcvmsg[2:])
			adressOfSite = append(adressOfSite, otherKey)
			l.Println(Nom, "Clé des autres : ", adressOfSite)

		} else if rcvmsg[:2] == "T:" {
			l.Println(Nom, "Nouvelle transaction reçu")
			newTransaction := ReceiveTransaction(rcvmsg[2:])
			if newTransaction.Verify(&blockChain) {
				pendingTransactions = append(pendingTransactions, newTransaction)
				l.Println(Nom, "Nouvelle transaction ajoutée")
			} else {
				l.Println(Nom, "Nouvelle transaction refusée")
			}

		} else if rcvmsg[:2] == "B:" {
			// vérifier la blockChain
			l.Println(Nom, "Nouveau bloc reçu")
			newBlock := ReceiveBlock(rcvmsg[2:])
			if len(blockChain.Chain) == 0 {
				//Bloc d'origine, il n'y a pas de vérification
				blockChain.AddBlock(newBlock)

			} else {
				if newBlock.VerifyBlock(blockChain) {
					blockChain.AddBlock(newBlock)
					l.Println(Nom, "Nouveau bloc ajouté")
					//l.Println(Nom, blockChain)
				} else {
					l.Println(Nom, "Nouveau bloc rejeté")
				}
			}
		} else if rcvmsg == "debutSC" {
			l.Println(Nom, "Autorisation de write")
			allowSC = true
		}
		mutex.Unlock()
		rcvmsg = ""
	}
}

var mutex = &sync.Mutex{}

func main() {
	flag.Parse()
	var rcvmsg string

	for {
		fmt.Scanln(&rcvmsg)
		if rcvmsg == "CONT:start" {
			break
		}
	}

	time.Sleep(time.Duration(1) * time.Second)
	l.Println(Nom, "Je commence mon initialisation")

	go sendInitialisation()
	go receive()
	for {
		time.Sleep(time.Duration(60) * time.Second)
	} // Pour attendre la fin des goroutines...*/
}
