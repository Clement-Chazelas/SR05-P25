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

var fieldsep = "$"
var keyvalsep = "~"

// Codes pour le terminal
var rouge string = "\033[1;31m"
var orange string = "\033[1;33m"
var vert string = "\033[1;32m"
var raz string = "\033[0;00m"

var pid = os.Getpid()
var stderr = log.New(os.Stderr, "", 0)

var Nom string

var MyId = -1
var NbSite = 3

const (
	MsgSender     string = "sdr"
	MsgCategory   string = "cat"
	MsgHorloge    string = "hrl"
	MsgType       string = "typ"
	MsgData       string = "dat"
	MsgEstampille string = "est"
	MsgColor 			string = "clr"
	app           string = "app"
	file          string = "file"
	snapshot      string = "snapshot"
)

var Sites []string

func MsgFormat(key string, val string) string {
	return fieldsep + keyvalsep + key + keyvalsep + val
}

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

func initialisation() {
	// Initialisation -> remplissage du tableau Sites avec les noms des autres controleurs
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

func main() {
	//Il ne faut pas oublier que le canal ctrleur -> app et ctlreur -> ctrl est le même
	//Il faut donc trier les messages à la lecture que ce soit coté ctrleur ou app
	//Pour ne pas traiter des messages dont on n'est pas le destinataire
	var rcvmsg string
	var pNom = flag.String("n", "controle", "Nom")
	flag.Parse()

	Nom = *pNom + "-" + strconv.Itoa(pid)
	initialisation()

	for {

		fmt.Scanln(&rcvmsg)

		//display_d("main", "réception de "+rcvmsg)

		rcvSdr := findval(rcvmsg, MsgSender)
		if rcvSdr == Nom {
			//Je ne traite pas un message que j'ai envoyé me revenant (anneau unidirectionnel)
			continue
		}

		rcvCat := findval(rcvmsg, MsgCategory)
		// Si champ Catégorie est présent -> le msg vient d'un autre controleur
		if rcvCat != "" {
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
				msgType := findval(rcvmsg, MsgType)
				switch msgType {
				case string(prepost):
					ReceivePrepostMessage(rcvmsg)
				case string(state):
					ReceiveStateMessage(rcvmsg)
				}

				fmt.Println(rcvmsg)

				break
			}
		} else {
			//Le msg vient de l'application

			if rcvmsg[:5] == "FILE:" {
				// Ce message concerne la section critique
				ReceiveSC(rcvmsg[5:])
			} else if rcvmsg[:5] == "SNAP:" {
				// Ce message concerne la snapshot
				//faire des trucs en rapport avec la snapshot
			} else if rcvmsg[:5] == "CONT:" {
				//Ignorer : ce message vient d'un autre controleur pour son application
				rcvmsg = ""
				continue
			} else {
				//Envoie du message de l'app aux autres controleurs
				newMessage := MsgFormat(MsgSender, Nom) +
					MsgFormat(MsgCategory, app) +
					MsgFormat(MsgData, rcvmsg) +
					MsgFormat(MsgColor, color) +
					MsgFormat(MsgHorloge, vectorClock)
				//Il faudra ajouter HVectorielle
				fmt.Println(newMessage)
			}
		}
		rcvmsg = ""

	}
}

func display_d(where string, what string) {
	stderr.Printf("%s + [%s] %-8.8s : %s\n%s", vert, Nom, where, what, raz)
}

func display_w(where string, what string) {

	stderr.Printf("%s * [%s] %-8.8s : %s\n%s", orange, Nom, where, what, raz)
}

func display_e(where string, what string) {
	stderr.Printf("%s ! [%s] %-8.8s : %s\n%s", rouge, Nom, where, what, raz)
}
