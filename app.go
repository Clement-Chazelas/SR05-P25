package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var rouge string = "\033[1;31m"
var orange string = "\033[1;33m"
var vert string = "\033[1;32m"
var magenta string = "\033[1;30m" // Bleu (Gras)
var cyan string = "\033[1;36m"    // Cyan (Gras)
var blanc string = "\033[1;37m"   // Blanc/Gris clair (Gras)

var raz string = "\033[0;00m"

var blockChain Blockchain

var adressOfSites []ecdsa.PublicKey
var nameOfSites []string

var pendingTransactions []Transaction

var sitePrivKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
var sitePubKey = sitePrivKey.PublicKey

var nbSite = 3
var allowSC = false

var Nom string

var stderr = log.New(os.Stderr, "", 0)
var verbose = false
var stdverb = log.New(io.Discard, "", 0)

func sendInitialisation() {
	isKeySent := false
	for {
		if !isKeySent {
			//J'envoie ma clé aux autres sites (une seule fois)
			mutex.Lock()

			stderr.Println(Nom, "J'envoie ma clé aux autres sites")
			fmt.Printf("K:%s%s\n", Nom, SendPublicKey(&sitePubKey))
			mutex.Unlock()
			isKeySent = true
			time.Sleep(time.Duration(1) * time.Second)
		}

		if len(adressOfSites) >= nbSite-1 && sitePubKey.X.Cmp(adressOfSites[0].X) == 1 && sitePubKey.X.Cmp(adressOfSites[1].X) == 1 {
			//j'ai reçu tt les clés + j'ai la clé la plus grande
			stderr.Println()
			stderr.Println(cyan, Nom, "Je suis l'initialisateur de la blockchain", raz)
			mutex.Lock()

			blockChain.InitBlockchain(append(adressOfSites, sitePubKey))
			fmt.Printf("B:%s\n", SendBlock(blockChain.GetLastBlock()))
			mutex.Unlock()
			stderr.Println(cyan, Nom, "Initialisation terminée", raz)
			go sendMain()
			break
		} else if len(blockChain.Chain) > 0 {
			// J'attends le premier bloc de la chaine
			stderr.Println(cyan, Nom, "Initialisation terminée", raz)
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
				stdverb.Println(Nom, "Initialisation d'un nouveau bloc")

				newBlock := InitBlock(pendingTransactions, blockChain.GetLastBlock().Hash, blockChain.GetLastBlock().UTXOs)
				newBlock.MineBlock()
				blockChain.AddBlock(newBlock)
				fmt.Printf("B:%s\n", SendBlock(blockChain.GetLastBlock()))

				allowSC = false
				isSCAsked = false
				fmt.Printf("FILE:finSC\n")
				stderr.Println(magenta, Nom, "Fin de Section critique", raz)
				stderr.Println()

				mutex.Unlock()
				pendingTransactions = []Transaction{}

				time.Sleep(time.Duration(5) * time.Second)
			} else if !isSCAsked {
				// On ne demande qu'une seule fois l'accès à la section critique
				fmt.Printf("FILE:demandeSC\n")
				isSCAsked = true
				stderr.Println(orange, Nom, "Demande Section critique", raz)

				time.Sleep(time.Duration(2) * time.Second)
			} else {
				// J'attends mon autorisation d'accès à la section critique, je continue d'envoyer des transac en attendant
				mutex.Lock()

				amount := mrand.Intn(10) + 1
				index := mrand.Intn(len(adressOfSites))

				newTransaction := InitTransaction(sitePubKey, adressOfSites[index], amount)
				newTransaction.Sign(sitePrivKey)
				fmt.Printf("T:%s\n", SendTransaction(&newTransaction))

				mutex.Unlock()

				stderr.Printf("%s %s Nouvelle transaction de %d coins envoyée à %s %s", blanc, Nom, amount, nameOfSites[index], raz)
				time.Sleep(time.Duration(5) * time.Second)
			}

		} else {
			//Je n'ai rien à miner donc création et envoi d'une transaction

			if allowSC {
				//La section critique avait été demandé mais mes transactions ont déjà été minés par un autre nœud
				allowSC = false
				isSCAsked = false
				fmt.Printf("FILE:finSC\n")
				stderr.Println(rouge, Nom, "Je n'ai pas de transaction en attente", raz)
				stderr.Println(magenta, Nom, "Fin de Section critique", raz)
				stderr.Println()
			}

			mutex.Lock()

			amount := mrand.Intn(10) + 1
			index := mrand.Intn(len(adressOfSites))

			newTransaction := InitTransaction(sitePubKey, adressOfSites[index], amount)
			newTransaction.Sign(sitePrivKey)
			fmt.Printf("T:%s\n", SendTransaction(&newTransaction))

			mutex.Unlock()

			stderr.Printf("%s %s Nouvelle transaction de %d coins envoyée à %s %s", blanc, Nom, amount, nameOfSites[index], raz)
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
			keyIndex := strings.Index(rcvmsg, "{")
			otherKey := ReceivePublicKey(rcvmsg[keyIndex:])

			adressOfSites = append(adressOfSites, otherKey)
			nameOfSites = append(nameOfSites, rcvmsg[2:keyIndex])

			stdverb.Println(Nom, "J'ai reçu", len(adressOfSites), "clé(s)")

		} else if rcvmsg[:2] == "T:" {
			stdverb.Println(Nom, "Nouvelle transaction reçu")
			newTransaction := ReceiveTransaction(rcvmsg[2:])
			if newTransaction.Verify(&blockChain) {
				pendingTransactions = append(pendingTransactions, newTransaction)
				stdverb.Println(Nom, "Nouvelle transaction ajoutée")
			} else {
				stderr.Println(rouge, Nom, "Nouvelle transaction refusée", raz)
			}

		} else if rcvmsg[:2] == "B:" {
			// vérifier la blockChain
			stdverb.Println(Nom, "Nouveau bloc reçu")
			newBlock := ReceiveBlock(rcvmsg[2:])
			if len(blockChain.Chain) == 0 {
				//Bloc d'origine, il n'y a pas de vérification
				blockChain.AddBlock(newBlock)

			} else {
				if newBlock.VerifyBlock(blockChain) {
					blockChain.AddBlock(newBlock)
					stdverb.Println(Nom, "Nouveau bloc ajouté")
					//On enlève de la liste de transactions celles présententes dans le bloc
					pendingTransactions = newBlock.updateTransactionsFromBlock(pendingTransactions)
					//l.Println(Nom, blockChain)
				} else {
					stderr.Println(rouge, Nom, "Nouveau bloc rejeté", raz)
				}
			}
		} else if rcvmsg == "debutSC" {
			stderr.Println()
			stderr.Println(vert, Nom, "J'entre dans la section critique", raz)
			allowSC = true
		}
		mutex.Unlock()
		rcvmsg = ""
	}
}

var mutex = &sync.Mutex{}

func main() {
	var rcvmsg string

	pNom := flag.String("n", "app", "Nom")
	flag.BoolVar(&verbose, "v", false, "Activer le mode verbose")
	flag.Parse()

	Nom = *pNom + "-" + strconv.Itoa(os.Getpid())
	if verbose {
		stdverb = log.New(os.Stderr, "", 0)
	}

	for {
		fmt.Scanln(&rcvmsg)
		if rcvmsg == "CONT:start" {
			// Attente du signal de départ du controleur
			break
		}
	}

	time.Sleep(time.Duration(1) * time.Second)
	stderr.Println(Nom, "Je commence mon initialisation")

	go sendInitialisation()
	go receive()
	for {
		time.Sleep(time.Duration(60) * time.Second)
	} // Pour attendre la fin des goroutines...*/
}
