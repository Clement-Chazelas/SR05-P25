package main

/*
Ce fichier contient les variables et les fonctions propres au fonctionnement d'un controleur d'application.
Le controleur joue un rôle de médiateur entre l'application (app) et les autres sites du réseau réparti.
Gère la diffusion des messages, les filtres de réception, implémente l'algorithme de la file d'attente répartie
ainsi que le déclenchement et la gestion de capture d'instantané (snapshot).
Le contrôleur s'occupe également d'envoyer à chaque modification une copie de la blockchain, de la file d'attente et
de la liste des contrôleurs au NET.
*/

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Caractères utilisés pour formater les messages entre les sites
var fieldsep = "$"
var keyvalsep = "~"

// Couleurs utilisées pour l'affichage dans le terminal
var rouge string = "\033[1;31m"
var orange string = "\033[1;33m"
var vert string = "\033[1;32m"
var raz string = "\033[0;00m"

var pid = os.Getpid()

// Récupération de la sortie erreur standard
var stderr = log.New(os.Stderr, "", 0)

// Nom du controleur, défini par l'utilisateur + pid du processus
var Nom string

// Nombre de sites dans la blockchain
var NbSite int

// Liste des noms des autres controleurs
var Sites []string

// Identifiant du controleur (= indice dans le tableau Sites)
var MyId = -1

// Constantes utilisées pour définir les champs de messages et certaines valeurs
const (
	MsgSender     string = "sdr"
	MsgCategory   string = "cat"
	MsgHorloge    string = "hrl"
	MsgType       string = "typ"
	MsgData       string = "dat"
	MsgEstampille string = "est"
	MsgColor      string = "clr"
	app           string = "app"
	file          string = "file"
	snapshot      string = "snapshot"
	newSite       string = "newsite"
	leave         string = "lve"
)

// MsgFormat construit une partie de message formatée avec une clé et une valeur
func MsgFormat(key string, val string) string {
	return fieldsep + keyvalsep + key + keyvalsep + val
}

// findval extrait la valeur correspondant à une clé dans un message formaté
// Renvoi une chaine vide si la clé n'est pas trouvée
func findval(msg string, key string) string {

	if len(msg) < 4 {
		display_w("findval", "message trop court : "+msg)
		return ""
	}

	tab_allkeyvals := strings.Split(msg[1:], fieldsep)

	for _, keyval := range tab_allkeyvals {
		tabkeyval := strings.Split(keyval[1:], keyvalsep)
		if tabkeyval[0] == key {
			return tabkeyval[1]
		}
	}
	return ""

}

// initialisation réalise l'échange des noms entre site.
// ELle permet de remplir la liste des noms des controleurs et attribue un ID unique à chacun
func initialisation() {
	var rcvmsg string

	// Mettre a jour fileAtt et vectorClock avec NbSite reçu
	fileAtt = make([]messageFile, NbSite)
	vectorClock = make([]int, NbSite)
	// J'ajoute mon propre nom au tableau
	Sites = append(Sites, Nom)
	// J'envoie mon nom aux autres controleurs
	fmt.Println(Nom)

	display_d("initialisation", "Début")

	for len(Sites) < NbSite {
		// Tant que je n'ai pas reçu tous les noms
		fmt.Scanln(&rcvmsg)

		if rcvmsg[:4] != "NET:" {
			//Ce message n'était pas à destination du controlleur
			rcvmsg = ""
			continue
		}

		rcvmsg = rcvmsg[4:]
		if rcvmsg == Nom {
			// J'ignore mon propre nom
			continue
		}

		// Ajout du nom reçu dans la liste
		Sites = append(Sites, rcvmsg)
	}

	//Le tableau Sites est trié par ordre alphabétique
	sort.Strings(Sites)
	// L'indice du Nom dans le tableau correspond à lid du site
	MyId = sort.SearchStrings(Sites, Nom)

	display_d("initialisation", fmt.Sprintf("Mon id est %d", MyId))

	display_d("initialisation", "Fin")

	// Envoie de la liste des contrôleurs au NET
	fmt.Printf("Controleurs:%s\n", strings.Join(Sites, ","))

	// Envoi du signal de départ à l'application
	fmt.Printf("CONT:start:%d\n", NbSite)

	return
}

// initialisationNouveauSite permet à un contrôleur de rejoindre un réseau de controleur déjà existant
func initialisationNouveauSite() {
	var rcvmsg string

	// Création de l'horloge vectorielle
	vectorClock = make([]int, NbSite)

	for {
		fmt.Scanln(&rcvmsg)

		// Le message n'était pas destinée au contrôleur
		if rcvmsg[:4] != "NET:" {
			rcvmsg = ""
			continue
		}
		// Suppression du préfix
		rcvmsg = rcvmsg[4:]

		// Reception de la liste des contrôleurs du NET
		if rcvmsg[:11] == "controleur:" {
			Sites = strings.Split(rcvmsg[11:], ",")
			Sites = append(Sites, Nom)
			MyId = sort.SearchStrings(Sites, Nom)
			// ID du dernier site ajouté
			newContIndex = MyId

		} else if rcvmsg[:6] == "queue:" {
			// Reception de la file d'attente envoyée par le NET
			fileAtt = receiveFileAtt(rcvmsg[6:])
			// Ajouter une case à la file d'attente (en fonction de mon ID)
			fileAtt = addSiteToFile(MyId)
			// Initialisation de mon estampille max(fileAttente) + 1
			for _, msg := range fileAtt {
				if msg.Date > estamp {
					estamp = msg.Date + 1
				}
			}
		} else if rcvmsg[:11] == "blockchain:" {
			// Reception de la blockchain envoyée par le NET
			localBlockchain = ReceiveBlockchain(rcvmsg[11:])
			break
		}

	}
	display_w("Initialisation", "fin")

	// Envoyer message pour indiquer son arrivée aux autres contrôleurs
	newMsg := MsgFormat(MsgSender, Nom) + MsgFormat(MsgCategory, newSite)
	fmt.Println(newMsg)

	//Lancement de l'initialisation de l'app et envoi de la blockchain
	fmt.Printf("CONT:start:%d\n", NbSite)
	fmt.Printf("CONT:blockchain:%s\n", SendBlockchain(localBlockchain.ToBlockchain()))

	fmt.Println("FinInit")
	return

}

// Fonction principale du controleur, elle lit les messages entrants et les traite dans une boucle infinie
func main() {
	var rcvmsg string
	var pNom = flag.String("n", "controle", "Nom")
	var pNouveauSite = flag.Bool("new", false, "Nouveau site")
	flag.Parse()

	// Récupération du nom donnée par l'utilisateur
	Nom = *pNom + "-" + strconv.Itoa(pid)

	fmt.Scanln(&rcvmsg)

	// Attente que le NET indique le départ du controleur
	for {
		fmt.Scanln(&rcvmsg)

		if len(rcvmsg) > 10 && rcvmsg[:10] == "NET:start:" {
			// Récupération du nombre de sites
			NbSite, _ = strconv.Atoi(rcvmsg[10:])
			break
		}
	}

	// Lancement de l'initialisation des contrôleurs
	if *pNouveauSite {
		initialisationNouveauSite()
	} else {
		initialisation()
	}

	for {

		fmt.Scanln(&rcvmsg)

		// L'application m'indique la fin
		if rcvmsg == "fin" {
			// Envoi d'un release pour empêcher le blocage de la file d'attente
			sendFileMessage(release)
			// Envoi d'un message indiquant mon départ
			newMsg := MsgFormat(MsgSender, Nom) + MsgFormat(MsgCategory, leave)
			fmt.Println(newMsg)
			// Arrêt de la boucle infinie
			break
		}

		if rcvmsg == "startSnapshot" {
			//J'ai reçu le signal pour démarrer la snapshot
			InitSnapshot()
			continue
		}

		// Le message ne débute pas par "NET:" ou "APP:"
		if rcvmsg[:4] != "NET:" && rcvmsg[:4] != "APP:" {
			// Ce message n'était pas à destination du controlleur
			rcvmsg = ""
			continue
		}

		// Suppresion du préfixe "NET:"
		rcvmsg = rcvmsg[4:]

		// Récupération de la catégorie du message
		rcvCat := findval(rcvmsg, MsgCategory)

		// Si champ Catégorie est présent, le message vient d'un autre controleur
		if rcvCat != "" {
			switch rcvCat {
			case app:
				// Traitement du message reçu par l'algorithme de snapshot
				ReceiveAppMessage(rcvmsg)

				rcvData := findval(rcvmsg, MsgData)
				//Envoi de la donnée reçue à l'application avec le préfixe "CONT:"
				fmt.Printf("CONT:%s\n", rcvData)

				break

			case file:
				// Traitement du message reçu par l'algorithme de la file d'attente
				ReceiveFileMessage(rcvmsg)
				// Envoi de la file d'attente au NET
				fmt.Printf("Queue:%s\n", sendFileAtt(fileAtt))
				break

			case snapshot:
				if initiator {
					// Je suis l'initiateur, je traite le message
					msgType := findval(rcvmsg, MsgType)
					switch msgType {

					case string(prepost):
						// Je traite le message de type prepost
						ReceivePrepostMessage(rcvmsg)

					case string(state):
						// Je traite le message de type state
						ReceiveStateMessage(rcvmsg)
					}
				}
				break

			// Le message indique l'arrivée d'un nouveau contrôleur
			case newSite:
				// Maj du nombre de site
				NbSite++
				// Récupération du nom du nouveau site et ajout à la liste
				newName := findval(rcvmsg, MsgSender)
				Sites = append(Sites, newName)
				sort.Strings(Sites)

				// Récupération de mon nouvel ID (ordre alphabétique)
				MyId = sort.SearchStrings(Sites, Nom)

				// Sauvegarde de l'ID du nouveau site
				newContIndex = sort.SearchStrings(Sites, newName)

				// Agrandissement de la file d'attente et de l'horloge vectorielle en fonction de l'ID du nouveau
				fileAtt = addSiteToFile(newContIndex)
				vectorClock = addSiteToClock(vectorClock, newContIndex)

				// On envoie la nouvelle liste des noms et la file d'attente au NET
				fmt.Printf("Controleurs:%s\n", strings.Join(Sites, ","))
				fmt.Printf("Queue:%s\n", sendFileAtt(fileAtt))

				display_d("Nouveau site", newName)
				break

			// Le message indique le départ d'un contrôleur
			case leave:
				// Maj du nombre de site
				NbSite--

				// Récupération du nom du site parant et de son ID
				quitSite := findval(rcvmsg, MsgSender)
				quitContIndex = sort.SearchStrings(Sites, quitSite)

				// Suppression du site partant de la liste des controleurs
				Sites = append(Sites[:quitContIndex], Sites[quitContIndex+1:]...)

				display_w("Depart", quitSite)
				//stderr.Println(Nom, "New site list", Sites)

				// Récupération de mon nouvel ID
				MyId = sort.SearchStrings(Sites, Nom)

				// Réduction de la file d'attente et de l'horloge vectorielle en fonction de l'ID du partant
				fileAtt = removeSiteFromFile(quitContIndex)
				vectorClock = removeSiteFromClock(vectorClock, quitContIndex)

				// On envoie la nouvelle liste des noms et la file d'attente au NET
				fmt.Printf("Controleurs:%s\n", strings.Join(Sites, ","))
				fmt.Printf("Queue:%s\n", sendFileAtt(fileAtt))

				// Temps d'attente pour laisser la modification des fifo
				time.Sleep(time.Duration(5) * time.Second)
				break
			}
		} else {
			//Le message vient de l'application
			if len(rcvmsg) < 5 {
				stderr.Println(Nom, "Message trop court", rcvmsg)
				rcvmsg = ""
				continue
			}
			if rcvmsg[:5] == "FILE:" {
				// Ce message concerne la section critique
				ReceiveSC(rcvmsg[5:])
			} else if rcvmsg[:5] == "SNAP:" {
				// Ce message contient la blockchain de l'application

				// Copie locale de la blockchain
				localBlockchain = ReceiveBlockchain(rcvmsg[5:])
				fmt.Printf("Blockchain:%s\n", rcvmsg[5:])
			} else if rcvmsg[:5] == "CONT:" {
				// Ignorer : ce message vient d'un autre controleur pour son application
				rcvmsg = ""
				continue
			} else {
				//Le message est à déstination des autres applications

				// Maj de l'horloge vectorielle
				vectorClock[MyId]++

				// Formatage du message pour l'envoi
				newMessage := MsgFormat(MsgSender, Nom) +
					MsgFormat(MsgCategory, app) +
					MsgFormat(MsgData, rcvmsg) +
					MsgFormat(MsgColor, myColor) +
					MsgFormat(MsgHorloge, ClockToStr(vectorClock))

				// Envoi du message
				fmt.Println(newMessage)
			}
		}
		rcvmsg = ""

	}
	// Le contrôleur est terminé
	stderr.Println(rouge, Nom, "Fin du programme", raz)
	time.Sleep(time.Duration(100) * time.Millisecond)
	// Indique au NET de s'arrêter à son tour
	fmt.Println("fin")
	for {
		// Lecture infinie pour ne pas bloquer la fifo
		fmt.Scanln(&rcvmsg)
		rcvmsg = ""
	}
}

// display_d affiche un message de debug en vert
func display_d(where string, what string) {
	stderr.Printf("%s + [%s] %-8.8s : %s\n%s", vert, Nom, where, what, raz)
}

// display_w affiche un message d'avertissement en orange
func display_w(where string, what string) {

	stderr.Printf("%s * [%s] %-8.8s : %s\n%s", orange, Nom, where, what, raz)
}

// display_e affiche une erreur en rouge
func display_e(where string, what string) {
	stderr.Printf("%s ! [%s] %-8.8s : %s\n%s", rouge, Nom, where, what, raz)
}
