// Joue le rôle de médiateur entre l'application (app) et les autres sites du réseau réparti
// Gère la diffusion des messages, les filtres de réception, l’initialisation de l’anneau,
// ainsi que le déclenchement et la gestion du protocole de snapshot distribué.

package main

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

// Initialisation des variables globales
var pid = os.Getpid()
var stderr = log.New(os.Stderr, "", 0)

var Nom string  // Nom complet du site (ex: C1-xxxxx)
var MyId = -1   // ID du site dans le tableau trié
var NbSite = 3  // Nombre total de sites dans l'anneau

// Constantes utilisées dans les messages
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
)

var Sites []string  // Liste triée des noms des contrôleurs

// MsgFormat construit une chaîne de message formattée avec une clé et une valeur
func MsgFormat(key string, val string) string {
	return fieldsep + keyvalsep + key + keyvalsep + val
}

// findval extrait la valeur correspondant à une clé dans un message formatté
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

// initialisation établit le tableau des sites et attribue un ID unique trié
func initialisation() {
	var rcvmsg string

	Sites = append(Sites, Nom)
	fmt.Println(Nom)

	display_d("initialisation", "Début")

	for len(Sites) < NbSite {
		fmt.Scanln(&rcvmsg)
		if rcvmsg == Nom {
			continue
		}
		Sites = append(Sites, rcvmsg)
		fmt.Println(rcvmsg)
	}

	//Le tableau Sites est trié par ordre alphabétique, l'indice du Nom dans le tableau = id du site
	sort.Strings(Sites)
	MyId = sort.SearchStrings(Sites, Nom)

	display_d("initialisation", fmt.Sprintf("Mon id est %d", MyId))

	display_d("initialisation", "Fin")

	fmt.Printf("CONT:start\n")
	return
}

// main boucle indéfiniment pour lire, traiter et relayer les messages entrants
func main() {
	var rcvmsg string
	var pNom = flag.String("n", "controle", "Nom")
	flag.Parse()

	Nom = *pNom + "-" + strconv.Itoa(pid)
	initialisation()

	for {
		fmt.Scanln(&rcvmsg)
		rcvSdr := findval(rcvmsg, MsgSender)

		if rcvSdr == Nom {
			continue //Je ne traite pas un message que j'ai envoyé me revenant (anneau unidirectionnel)
		}

		if rcvmsg == "startSnapshot" {
			//J'ai reçu le signal pour démarrer la snapshot
			InitSnapshot()
			continue
		}

		rcvCat := findval(rcvmsg, MsgCategory)

		// Si champ Catégorie est présent -> le msg vient d'un autre controleur
		if rcvCat != "" {
			// message inter-contrôleur
			switch rcvCat {
			case app:
				// Maj horloge vectorielle
				ReceiveAppMessage(rcvmsg)
				rcvData := findval(rcvmsg, MsgData)
				//Envoi de la donnée reçue à l'application
				fmt.Printf("CONT:%s\n", rcvData)
				// Renvoyer le message reçu (car anneau unidirectionnel)
				fmt.Println(rcvmsg)
				break

			case file:
				ReceiveFileMessage(rcvmsg)

				// Renvoyer le message reçu (car anneau unidirectionnel)
				fmt.Println(rcvmsg)
				break

			case snapshot:
				if initiator {
					msgType := findval(rcvmsg, MsgType)
					switch msgType {
					case string(prepost):
						ReceivePrepostMessage(rcvmsg)
					case string(state):
						ReceiveStateMessage(rcvmsg)
						stderr.Println(localSnapshot)
					}
				} else {
					// Je ne suis pas l'initiateur, je relai la snapshot
					fmt.Println(rcvmsg)
				}
				break
			}
		} else {
			//Le msg vient de l'application
			if rcvmsg[:5] == "FILE:" {
				// Ce message concerne la section critique
				ReceiveSC(rcvmsg[5:])
			} else if rcvmsg[:5] == "SNAP:" {
				// Ce message concerne la snapshot
				localBlockchain = ReceiveBlockchain(rcvmsg[5:])
			} else if rcvmsg[:5] == "CONT:" {
				//Ignorer : ce message vient d'un autre controleur pour son application
				rcvmsg = ""
				continue
			} else {
				//Envoie du message de l'app aux autres controleurs
				vectorClock[MyId]++ //Mise à jour de l'horloge vectorielle
				newMessage := MsgFormat(MsgSender, Nom) +
					MsgFormat(MsgCategory, app) +
					MsgFormat(MsgData, rcvmsg) +
					MsgFormat(MsgColor, myColor) +
					MsgFormat(MsgHorloge, ClockToStr(vectorClock))
				//Il faudra ajouter HVectorielle
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
