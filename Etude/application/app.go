package main

/*
Ce fichier contient les variables et les fonctions propres au fonctionnement de l'application.
L'application est séquentielle, les actions de lecture et d'écriture sont atomiques et la lecture est asynchrone.
Elle simule le fonctionnement d'une blockchain, incluant les vérifications nécessaires à sa sécurité.
La différence principale avec une blockchain classique est que l'accès en écriture (minage) est non-concurrent.
*/

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

// Listes des couleurs pour l'affichage dans la console
var noir string = "\033[1;30m"
var rouge string = "\033[1;31m"
var vert string = "\033[1;32m"
var orange string = "\033[1;33m"
var magenta string = "\033[1;35m"
var cyan string = "\033[1;36m"
var blanc string = "\033[1;37m"
var raz string = "\033[0;00m"

// Copie locale de la blockchain répartie
var blockChain Blockchain

// Liste des adresses des autres sites et leurs noms.
// L'indice d'un même site est égal dans les deux listes.
var adressOfSites []ecdsa.PublicKey
var nameOfSites []string

// Liste des transactions reçues en attente d'être minées
var pendingTransactions []Transaction

// Clé privée et clé publique de l'application
var sitePrivKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
var sitePubKey = sitePrivKey.PublicKey

// Nombre de sites dans la blockchain
var nbSite int

// Booléen indiquant si l'accès à la section critique a été accordé par le controleur
var allowSC = false

// Booléen indiquant si le mode verbose est activé
var verbose = false

// Nom du site, défini par l'utilisateur + pid du processus
var Nom string

// Sortie d'erreur standard et sortie de verbose (rejetée par défaut)
var stderr = log.New(os.Stderr, "", 0)
var stdverb = log.New(io.Discard, "", 0)

var prefix = "APP:"

var mutex = &sync.Mutex{}

// sendInitialisation est la fonction d'écriture permettant de réaliser l'initialisation de l'application.
// Elle consiste en l'échange des clés et des noms entre les différents sites, et l'initialisation du premier block.
func sendInitialisation(stop chan struct{}, fin chan struct{}) {
	// Indique si l'application a déjà envoyé sa clé aux autres sites.
	isKeySent := false
	for {
		// Je n'ai pas envoyé ma clé
		if !isKeySent {
			mutex.Lock()

			stderr.Println(Nom, "J'envoie ma clé aux autres sites")
			// Envoi de la clé et du nom du site avec le préfixe "K:"
			fmt.Printf("%sK:%s%s\n", prefix, Nom, SendPublicKey(&sitePubKey))

			mutex.Unlock()
			isKeySent = true
			time.Sleep(time.Duration(1) * time.Second)
		}
		// J'ai récupéré toutes les clés des autres sites et ma clé est la plus grande
		if len(adressOfSites) >= nbSite-1 && isTheBiggestKey(sitePubKey, adressOfSites) {
			// Je deviens initiateur de la blockchain
			stderr.Println()
			stderr.Println(cyan, Nom, "Je suis l'initialisateur de la blockchain", raz)
			mutex.Lock()

			// Initialisation du premier block
			blockChain.InitBlockchain(append(adressOfSites, sitePubKey))

			// Envoi du premier block aux autres sites avec le préfixe "B:"
			fmt.Printf("%sB:%s\n", prefix, SendBlock(blockChain.GetLastBlock()))

			// Envoi de la nouvelle blockchain au contrôleur pour qu'il possède l'état local
			fmt.Printf("%sSNAP:%s\n", prefix, SendBlockchain(blockChain))

			mutex.Unlock()
			stderr.Println(cyan, Nom, "Initialisation terminée", raz)
			// Lancement de la goroutines d'écriture principale
			go sendMain(stop, fin)
			break
		} else if len(blockChain.Chain) > 0 {
			// Je ne suis pas l'initiateur, j'attends le premier bloc de la blockchain

			stderr.Println(cyan, Nom, "Initialisation terminée", raz)
			// Lancement de la goroutines d'écriture principale
			go sendMain(stop, fin)
			break
		}

		time.Sleep(time.Duration(1) * time.Second)
	}
}

// sendInitialisationNouveauSite permet à une nouvelle application de rejoindre un réseau d'application
// Récupère la blockchain transmise par le contrôleur et envoi sa clé dans le réseau.
func sendInitialisationNouveauSite(stop chan struct{}, fin chan struct{}) {
	var rcvmsg string

	for {
		fmt.Scanln(&rcvmsg)

		// Le message ne m'était pas destiné
		if rcvmsg[:5] != "CONT:" {
			rcvmsg = ""
			continue
		}

		rcvmsg = rcvmsg[5:]
		// Récupération de la blockchain envoyée par le contrôleur
		if rcvmsg[:11] == "blockchain:" {
			rcvBlockchain := ReceiveBlockchain(rcvmsg[11:])
			blockChain = rcvBlockchain.ToBlockchain()
			stderr.Println(Nom, "Initalisation terminée")
			break
		}
	}

	stderr.Println(Nom, "J'envoie ma clé aux autres sites")
	// Envoi de la clé et du nom du site avec le préfixe "K:"
	fmt.Printf("%sK:%s%s\n", prefix, Nom, SendPublicKey(&sitePubKey))

	time.Sleep(time.Duration(1) * time.Second)

	// Démarrage des goroutines de fonctionnement classique
	go sendMain(stop, fin)
	go receive(stop)

}

// sendMain est la fonction d'écriture principale de l'application.
// Elle mine un block lorsque c'est possible, et envoie une transaction sinon.
// L'accès en écriture à la blockchain respecte l'algorithme de la file d'attente répartie.
// Les transactions sont envoyées à intervalle aléatoire de temps compris entre 1 et 10 secondes.
func sendMain(stop chan struct{}, fin chan struct{}) {
	// Booléen indiquant si l'application a réalisé une demande d'accès à la section critique
	isSCAsked := false
	for {
		select {
		// 	Receive indique l'arrêt des envois
		case <-stop:
			// Fermeture du canal de fin du programme
			close(fin)
			return
		// Sinon
		default:
			// Si la liste des transactions en attente est non vide
			if len(pendingTransactions) > 0 {

				if allowSC {
					// J'ai l'autorisation d'accès à la section critique
					mutex.Lock()

					stderr.Println()
					stderr.Println(vert, Nom, "J'entre dans la section critique", raz)

					stdverb.Println(Nom, "Initialisation d'un nouveau bloc")
					maxTransac := len(pendingTransactions)
					if maxTransac > 4 {
						maxTransac = 4
					}
					stderr.Printf(" %s%s Ajout des transactions : %sau bloc%s\n", cyan, Nom, printTransactionsId(pendingTransactions[:maxTransac]), raz)

					newBlock := InitBlock(pendingTransactions[:maxTransac], blockChain.GetLastBlock().Hash, blockChain.GetLastBlock().UTXOs)
					newBlock.MineBlock()
					blockChain.AddBlock(newBlock)
					//Envoi du block miné aux autres sites
					fmt.Printf("%sB:%s\n", prefix, SendBlock(blockChain.GetLastBlock()))
					//Envoi de la nouvelle blockchain au contrôleur pour save l'état local
					fmt.Printf("%sSNAP:%s\n", prefix, SendBlockchain(blockChain))

					allowSC = false
					isSCAsked = false
					fmt.Printf("%sFILE:finSC\n", prefix)
					stderr.Println(noir, Nom, "Fin de Section critique", raz)
					stderr.Println()

					mutex.Unlock()
					pendingTransactions = pendingTransactions[maxTransac:]

					time.Sleep(time.Duration(2) * time.Second)
				} else if !isSCAsked {
					// Je n'ai pas accès à la section critique, je demande l'autorisation (une unique fois)
					mutex.Lock()
					fmt.Printf("%sFILE:demandeSC\n", prefix)
					isSCAsked = true
					stderr.Println(orange, Nom, "Demande Section critique", raz)
					mutex.Unlock()
					time.Sleep(time.Duration(2) * time.Second)

				} else {
					// En attendant l'autorisation d'accès à la section critique, j'envoie une transaction
					mutex.Lock()

					// Je ne connais personne, je ne génère pas de transaction
					if len(adressOfSites) == 0 {
						mutex.Unlock()
						time.Sleep(time.Duration(1) * time.Second)
						continue
					}

					if blockChain.GetLastBlock().UTXOs.FindByKey(sitePubKey) == nil ||
						blockChain.GetLastBlock().UTXOs.FindByKey(sitePubKey).Amount < 10 {
						mutex.Unlock()
						time.Sleep(time.Duration(1) * time.Second)
						continue
					}

					// Montant et déstinataire choisi aléatoirement
					amount := mrand.Intn(10) + 1

					index := mrand.Intn(len(adressOfSites))

					//Initialisation de la transaction et signature
					newTransaction := InitTransaction(sitePubKey, adressOfSites[index], amount)
					newTransaction.Sign(sitePrivKey)

					//Envoi de la transaction avec le préfixe "T:"
					fmt.Printf("%sT:%s\n", prefix, SendTransaction(&newTransaction))

					mutex.Unlock()

					stderr.Printf("%s %s Nouvelle transaction de %d coins envoyée à %s ; ID=%d %s", blanc, Nom, amount, nameOfSites[index], newTransaction.Id, raz)
					time.Sleep(time.Duration(amount+5) * time.Second)
				}

			} else {
				// Je n'ai pas de transactions en attente, donc création et envoi d'une transaction
				mutex.Lock()

				if allowSC {
					//La section critique avait été demandé, mais mes transactions en attentes ont déjà été minées par un autre site
					stderr.Println()
					stderr.Println(vert, Nom, "J'entre dans la section critique", raz)

					// Réinitialisation des booléens de sections critiques
					allowSC = false
					isSCAsked = false

					// Envoi au controleur d'un message indiquant la fin de l'accès à la section critique
					fmt.Printf("%sFILE:finSC\n", prefix)

					stderr.Println(rouge, Nom, "Je n'ai pas de transaction en attente", raz)
					stderr.Println(noir, Nom, "Fin de Section critique", raz)
					stderr.Println()
				}

				// Je ne connais personne, je ne génère pas de transaction
				if len(adressOfSites) == 0 {
					mutex.Unlock()
					time.Sleep(time.Duration(1) * time.Second)

					continue
				}

				if blockChain.GetLastBlock().UTXOs.FindByKey(sitePubKey) == nil ||
					blockChain.GetLastBlock().UTXOs.FindByKey(sitePubKey).Amount < 10 {
					mutex.Unlock()
					time.Sleep(time.Duration(1) * time.Second)
					continue
				}

				// Montant et déstinataire choisi aléatoirement
				amount := mrand.Intn(10) + 1
				index := mrand.Intn(len(adressOfSites))

				//Initialisation de la transaction et signature
				newTransaction := InitTransaction(sitePubKey, adressOfSites[index], amount)
				newTransaction.Sign(sitePrivKey)

				//Envoi de la transaction avec le préfixe "T:"
				fmt.Printf("%sT:%s\n", prefix, SendTransaction(&newTransaction))

				mutex.Unlock()

				stderr.Printf("%s %s Nouvelle transaction de %d coins envoyée à %s ; ID=%d %s", blanc, Nom, amount, nameOfSites[index], newTransaction.Id, raz)
				time.Sleep(time.Duration(amount+5) * time.Second)
			}
		}
	}

}

// receive est la fonction de lecture de l'application.
// Elle reçoit les messages, les analyse et réalise les actions nécessaires.
func receive(stop chan struct{}) {
	var rcvmsg string

	for {
		fmt.Scanln(&rcvmsg)
		mutex.Lock()

		// Réception du message de fin
		if rcvmsg == "fin" {
			mutex.Unlock()
			stderr.Println(Nom, "Signal de fin reçu")
			// Fermeture de canal stop pour indiquer l'arrêt à la goroutine send
			close(stop)
			return
		}

		// Le message ne débute pas par "CONT:"
		if rcvmsg[:5] != "CONT:" {
			//Ce message n'était pas à destination de l'application
			mutex.Unlock()
			rcvmsg = ""
			continue
		}

		// Suppresion du préfixe "CONT:"
		rcvmsg = rcvmsg[5:]

		if rcvmsg[:2] == "K:" {
			// Le message reçu est une clé

			// Récupération de la clé (débutant par '{') et du nom du site
			keyIndex := strings.Index(rcvmsg, "{")
			rcvKey := ReceivePublicKey(rcvmsg[keyIndex:])

			adressOfSites = append(adressOfSites, rcvKey)
			nameOfSites = append(nameOfSites, rcvmsg[2:keyIndex])

			// Affichage si mode verbose
			stdverb.Println(Nom, "J'ai reçu", len(adressOfSites), "clé(s)")

		} else if rcvmsg[:2] == "T:" {
			if len(blockChain.Chain) == 0 {
				stderr.Println(Nom, "Transaction alors que chaine vide", rcvmsg)
				mutex.Unlock()
				rcvmsg = ""
				continue
			}
			// Le message reçu est une transaction
			stdverb.Println(Nom, "Nouvelle transaction reçu")

			// Récuparation de la transaction
			newTransaction := ReceiveTransaction(rcvmsg[2:])

			// Vérification de la transaction et ajout dans la liste des transactions en attente
			if newTransaction.Verify(&blockChain) {
				pendingTransactions = append(pendingTransactions, newTransaction)
				stdverb.Println(Nom, "Nouvelle transaction ajoutée")
			} else {
				stderr.Println(rouge, Nom, "Nouvelle transaction refusée", raz)
			}

		} else if rcvmsg[:2] == "B:" {
			// Le message reçu est un block
			stdverb.Println(Nom, "Nouveau bloc reçu")

			// Récupération du block
			newBlock := ReceiveBlock(rcvmsg[2:])

			if len(blockChain.Chain) == 0 {
				// Bloc d'origine, il n'y a pas de vérification
				blockChain.AddBlock(newBlock)

				// Envoi de la nouvelle blockchain au contrôleur pour qu'il possède l'état local
				fmt.Printf("%sSNAP:%s\n", prefix, SendBlockchain(blockChain))

			} else {
				// Sinon vérification du bloc et ajout dans la blockchain
				if newBlock.VerifyBlock(blockChain) {
					blockChain.AddBlock(newBlock)
					stdverb.Println(Nom, "Nouveau bloc ajouté")

					// Enlève de la liste des transactions en attentes celles présentes dans le block
					pendingTransactions = newBlock.updateTransactionsFromBlock(pendingTransactions)

					// Envoi de la nouvelle blockchain au contrôleur pour qu'il possède l'état local
					fmt.Printf("%sSNAP:%s\n", prefix, SendBlockchain(blockChain))

				} else {
					stderr.Println(rouge, Nom, "Nouveau bloc rejeté", raz)
				}
			}

		} else if rcvmsg == "debutSC" {
			// Le message reçu est une autorisation du controleur concernant la section critique
			allowSC = true
		}
		mutex.Unlock()
		rcvmsg = ""
	}
}

// Fonction principale de l'application, lancée à l'exécution du programme.
func main() {
	var rcvmsg string
	stop := make(chan struct{}) // Canal indiquant l'arrêt de l'envoi de message
	fin := make(chan struct{})  // Canal indiquant l'arrêt de l'application

	// Récupération du nom donnée par l'utilisateur et du mode verbose.
	pNom := flag.String("n", "app", "Nom")
	pNouveauSite := flag.Bool("new", false, "Nouveau site")
	flag.BoolVar(&verbose, "v", false, "Activer le mode verbose")
	flag.Parse()

	// Initialisation du nom du site en ajoutant le pid du processus.
	Nom = *pNom + "-" + strconv.Itoa(os.Getpid())
	// Activation du mode verbose si demandé
	if verbose {
		// stdverb est redirigé vers la sortie erreur standard
		stdverb = log.New(os.Stderr, "", 0)
	}

	// Attente que le controleur indique le départ de l'application
	for {
		fmt.Scanln(&rcvmsg)

		if len(rcvmsg) > 11 && rcvmsg[:11] == "CONT:start:" {
			// Le controleur a terminé son initialisation
			// Récupération du nombre de sites
			nbSite, _ = strconv.Atoi(rcvmsg[11:])
			break
		}
	}

	time.Sleep(time.Duration(1) * time.Second)
	stderr.Println(Nom, "Je commence mon initialisation")

	// Démarrage des goroutines de lecture et d'écriture.
	if *pNouveauSite {
		go sendInitialisationNouveauSite(stop, fin)
	} else {
		go receive(stop)
		go sendInitialisation(stop, fin)
	}

	// Attente bloquante de la fin des goroutines
	<-fin
	stderr.Println("\n", rouge, Nom, "Fin de l'application", raz)
	fmt.Println("fin")

	for {
		// Lecture infinie pour ne pas bloquer la fifo
		fmt.Scanln(&rcvmsg)
		rcvmsg = ""
	}
}
