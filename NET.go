package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	_ "sort"
	"strconv"
	"strings"
	"time"
)

var fieldsep = "§"
var keyvalsep = "£"

const (
	MsgSender      = "sdr"
	MsgDestination = "dest"
	MsgCategory    = "cat"
	MsgData        = "dat"
	MsgPath        = "pth"
	MsgType        = "typ"
	electionInit   = "eli"
	controleur     = "ctr"
)

var (
	pNom        = flag.String("n", "NET", "Nom du noeud")
	Nom         string
	pid         = os.Getpid()
	stderr      = log.New(os.Stderr, "", 0)
	MyId        = pid
	NbVoisins   = -1
	NbSites     = -1
	ListVoisins []int
)

func MsgFormat(key, val string) string {
	return fieldsep + keyvalsep + key + keyvalsep + val
}

func findval(msg, key string) string {
	if len(msg) < 4 {
		return ""
	}
	tab_allkeyvals := strings.Split(msg[1:], fieldsep)
	for _, keyval := range tab_allkeyvals {
		tabkeyval := strings.Split(keyval[1:], keyvalsep)
		if len(tabkeyval) == 2 && tabkeyval[0] == key {
			return tabkeyval[1]
		}
	}
	return ""
}

func initialisation() {

	fmt.Println(MyId)

	// Lecture des autres NET pour initialisation
	var received string
	heureDebut := time.Now()
	deadline := heureDebut.Add(5 * time.Second)
	for time.Now().Before(deadline) {
		fmt.Scanln(&received)
		idVoisin, _ := strconv.Atoi(received)
		ListVoisins = append(ListVoisins, idVoisin)
	}
	NbVoisins = len(ListVoisins)

	DemarrerElection()

	// Tant que je n'ai pas gagné l'élection (je n'ai pas fini de compter le nb de sites)
	for NbSites == -1 {
		fmt.Scanln(&received)

		msgCat := findval(received, MsgCategory)
		switch msgCat {
		case electionInit:
			recevoirMessageElection(received)
		}
	}

	if win {
		// J'ai remporté l'élection, je transmets le nb de site à mes enfants
		envoyerA(nombreSites, strconv.Itoa(NbSites), enfants)
	}
	stderr.Println("Fin de l'initialisation")
	fmt.Printf("start:%s\n",NbSites)
}

func majHistorique(msg string) string {
	hist := strToIntTab(findval(msg, MsgPath))
	hist[0] = hist[1]
	hist[1] = MyId
	newMsg := MsgFormat(MsgSender, findval(msg, MsgSender)) + MsgFormat(MsgCategory, findval(msg, MsgCategory)) +
		MsgFormat(MsgData, findval(msg, MsgData)) + MsgFormat(MsgPath, intTabToStr(hist))
	return newMsg
}

func main() {
	flag.Parse()
	Nom = *pNom + "-" + strconv.Itoa(pid)
	initialisation()

	var rcvmsg string
	for {
		fmt.Scanln(&rcvmsg)

		if rcvmsg[:5] == "CONT:" || rcvmsg[:4] == "NET:" {
			//Ce message n'était pas à destination du NET
			rcvmsg = ""
			continue
		}

		// Traitement des messages
		rcvCat := findval(rcvmsg, MsgCategory)
		if rcvCat != "" {
			// Le message vient d'un autre NET
			msgSdr := findval(rcvmsg, MsgSender)
			sdrId, _ := strconv.Atoi(msgSdr)

			if sdrId != parent || !slices.Contains(enfants, sdrId) {
				// Si le message ne vient pas de mon parent ou de mes enfants, je le rejette
				continue
			}

			msgHist := findval(rcvmsg, MsgPath)
			hist := strToIntTab(msgHist)

			if slices.Contains(hist, MyId) {
				// j'ai déjà traité ce message
				continue
			}

			switch rcvCat {
			case controleur:

				rcvData := findval(rcvmsg, MsgData)
				//Envoi de la donnée reçue au controleur avec le préfixe "NET:"
				fmt.Printf("NET:%s\n", rcvData)

				//Relai du message dans le réseau en mettant à jour l'historique
				fmt.Println(majHistorique(rcvmsg))
				break
			}

		} else {
			// Le message vient de mon controleur
			newMessage := MsgFormat(MsgSender, strconv.Itoa(MyId)) +
				MsgFormat(MsgCategory, controleur) +
				MsgFormat(MsgPath, intTabToStr([]int{MyId, MyId})) +
				MsgFormat(MsgData, rcvmsg)
			fmt.Println(newMessage)
		}
	}
}
