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

var pNom = flag.String("n", "controle", "Nom")
var Nom = *pNom + "-" + strconv.Itoa(pid)

var MyId = -1
var NbSite = 3

const (
	MsgSender     string = "sdr"
	MsgCategory   string = "cat"
	MsgHorloge    string = "hrl"
	MsgType       string = "typ"
	MsgData       string = "dat"
	MsgEstampille string = "est"
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

	sep := msg[0:1]
	tab_allkeyvals := strings.Split(msg[1:], sep)

	for _, keyval := range tab_allkeyvals {
		equ := keyval[0:1]
		tabkeyval := strings.Split(keyval[1:], equ)
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

	display_d("initialisation", "Fin")

	//Le tableau Sites est trié par ordre alphabétique, l'indice du Nom dans le tableau = id du site
	sort.Strings(Sites)
	MyId = sort.SearchStrings(Sites, Nom)

	display_d("initialisation", strconv.Itoa(MyId))

	fmt.Printf("CONT:start\n")
	return
}

func main() {
	//Il ne faut pas oublier que le canal ctrleur -> app et ctlreur -> ctrl est le même
	//Il faut donc trier les messages à la lecture que ce soit coté ctrleur ou app
	//Pour ne pas traiter des messages dont on n'est pas le destinataire
	var rcvmsg string

	flag.Parse()

	initialisation()

	for {

		fmt.Scanln(&rcvmsg)

		display_d("main", "réception de "+rcvmsg)

		rcvSdr := findval(rcvmsg, MsgSender)
		if rcvSdr == Nom {
			continue
		}

		rcvCat := findval(rcvmsg, MsgCategory)
		// Si champ Catégorie est présent -> le msg vient d'un autre controleur
		if rcvCat != "" {
			switch rcvCat {
			case app:
				// Maj horloge vectorielle
				rcvData := findval(rcvmsg, MsgData)
				//Envoi de la donnée reçue à l'application
				fmt.Printf("CONT:%s\n", rcvData)
				fmt.Println(rcvmsg)
				break

			case file:
				ReceiveFileMessage(rcvmsg)
				// Peut-être renvoyer le message reçu (si anneau unidirectionnel)
				fmt.Println(rcvmsg)
				break

			case snapshot:
				//Appel des fonctions spécifiques
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
				newMessage := MsgFormat(MsgSender, Nom) + MsgFormat(MsgCategory, app) + MsgFormat(MsgData, rcvmsg)
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
