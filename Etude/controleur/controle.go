package main

/*
Ce fichier contient les variables et les fonctions propres au fonctionnement d'un controleur d'application.
Le controleur joue un rôle de médiateur entre l'application (app) et les autres sites du réseau réparti.
Gère la diffusion des messages, les filtres de réception, implémente l'algorithme de la file d'attente répartie
ainsi que le déclenchement et la gestion de capture d'instantané (snapshot).
Ce programme est fait pour fonctionner dans un anneau unidirectionnel. Chaque message reçu est donc relayé au site suivant.
Pour éviter les doublons, le controleur ne traite pas les messages qu'il a lui-même envoyés.
*/

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
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
		// Je relaie le message reçu au site suivant
		//fmt.Println(rcvmsg)
	}

	//Le tableau Sites est trié par ordre alphabétique
	sort.Strings(Sites)
	// L'indice du Nom dans le tableau correspond à lid du site
	MyId = sort.SearchStrings(Sites, Nom)

	display_d("initialisation", fmt.Sprintf("Mon id est %d", MyId))

	display_d("initialisation", "Fin")

	fmt.Printf("Controleurs:%s\n", strings.Join(Sites, ","))
	// Envoi du signal de départ à l'application
	fmt.Printf("CONT:start:%d\n", NbSite)

	return
}

func initialisationNouveauSite() {
	var rcvmsg string

	for {
		fmt.Scanln(&rcvmsg)

		if rcvmsg[:4] != "NET:" {
			rcvmsg = ""
			continue
		}

		rcvmsg = rcvmsg[4:]

		if strings.HasPrefix(rcvmsg, "controller:") {
			Sites = append(Sites, rcvmsg[11:])
		} else if strings.HasPrefix(rcvmsg, "blockchain:") {
			localBlockchain = ReceiveBlockchain(rcvmsg[11:])
		} else if strings.HasPrefix(rcvmsg, "queue:") {
			//fileAtt = rcvmsg[6:]
			//Faire fonction pourenvoyer et recevoir file d'attnte en JSON
		}

		break
	}

	// Envoyer message de nouveau
	newMsg := MsgFormat(MsgSender, Nom) + MsgFormat(MsgCategory, newSite)
	fmt.Println(newMsg)
	//Lancement de l'initialisation de l'app et envoi des infos nécessaires
	fmt.Printf("CONT:start:%d\n", NbSite)
	fmt.Printf("CONT:blockchain:%s\n", localBlockchain)

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
			NbSite, _ = strconv.Atoi(rcvmsg[10:])
			break
		}
	}
	// Lancement de l'initialisation des controleurs
	if *pNouveauSite {
		initialisationNouveauSite()
	} else {
		initialisation()
	}

	for {

		fmt.Scanln(&rcvmsg)

		// Le message ne débute pas par "NET:"
		if rcvmsg[:4] != "NET:" && rcvmsg[:4] != "APP:" {
			//Ce message n'était pas à destination du controlleur
			rcvmsg = ""
			continue
		}

		// Suppresion du préfixe "NET:"
		rcvmsg = rcvmsg[4:]

		if rcvmsg == "startSnapshot" {
			//J'ai reçu le signal pour démarrer la snapshot
			InitSnapshot()
			continue
		}

		if rcvmsg[:8] == "newCont:" {

		}

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
			case newSite:
				NbSite++
				newName := findval(rcvmsg, MsgSender)
				Sites = append(Sites, newName)
				sort.Strings(Sites)
				MyId = sort.SearchStrings(Sites, Nom)

				newContIndex = sort.SearchStrings(Sites, newName)
				fileAtt = addSiteToFile(newContIndex)
				vectorClock = addSiteToClock(vectorClock, newContIndex)

				fmt.Printf("Controleurs:%s\n", strings.Join(Sites, ","))
				// Envoyer file att
				continue
			}
		} else {
			//Le message vient de l'application

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
