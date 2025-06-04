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

const (
	MsgCategory   = "cat"
	MsgData       = "dat"
	MsgPath       = "pth"
	MsgType       = "typ"
	MsgInit       = "init"
	MsgPropagation = "prop"
	MsgDestination = "dest"

	ToCtrl = "CTRL:"
	ToNET  = "NET:"
)

var (
	pNom     = flag.String("n", "NET", "Nom du noeud")
	Nom      string
	pid      = os.Getpid()
	stderr   = log.New(os.Stderr, "", 0)
	Parent   string
	Children []string
	MyId     int = -1
	Peers    = make(map[string]bool)
	Historique []string // derniers ids qui ont traité un message
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
	Nom = *pNom + "-" + strconv.Itoa(pid)
	fmt.Println(Nom)

	// Lecture des autres NET pour initialisation
	var received string
	Peers[Nom] = true
	for len(Peers) < 3 { // À adapter selon NbNet souhaité
		fmt.Scanln(&received)
		if received == Nom || Peers[received] {
			continue
		}
		Peers[received] = true
		fmt.Println(received)
	}

	// Création de l’arborescence
	names := []string{}
	for k := range Peers {
		names = append(names, k)
	}
	sort.Strings(names)
	for i, name := range names {
		if name == Nom {
			MyId = i
		}
	}
	// Choix du parent et enfants
	if MyId > 0 {
		Parent = names[(MyId-1)/2]
	}
	left := 2*MyId + 1
	right := 2*MyId + 2
	if left < len(names) {
		Children = append(Children, names[left])
	}
	if right < len(names) {
		Children = append(Children, names[right])
	}

	display_d("initialisation", fmt.Sprintf("Id: %d, Parent: %s, Children: %v", MyId, Parent, Children))
	fmt.Printf("NET:init:%d:%s\n", MyId, Nom)
}


func dejaTraite(msg string) bool {
	path := findval(msg, MsgPath)
	if path == "" {
		return false
	}
	for _, id := range strings.Split(path, ",") {
		if id == strconv.Itoa(MyId) {
			return true
		}
	}
	return false
}

func ajouterHistorique(id int) {
	if len(Historique) >= 2 {
		Historique = Historique[1:]
	}
	Historique = append(Historique, strconv.Itoa(id))
}


func traiterInit(msg string) {
	display_d("init", "Traitement vague d’init")
	transmettreAuxEnfants(msg)
}

func transmettreAuxEnfants(msg string) {
	for _, child := range Children {
		newMsg := msg + MsgFormat(MsgPath, strconv.Itoa(MyId))
		fmt.Printf("NET:%s\n", newMsg)
	}
}

func versControleur(msg string) {
	fmt.Printf("CTRL:%s\n", msg)
}



func main() {
	flag.Parse()
	initialisation()

	var msg string
	for {
		fmt.Scanln(&msg)

		if findval(msg, MsgSender) == Nom {
			continue
		}

		// Empêcher les re-traitements
		if dejaTraite(msg) {
			display_d("main", "Message déjà traité")
			continue
		}
		ajouterHistorique(MyId)

		// Traitement des messages
		rcvCat := findval(msg, MsgCategory)

		switch rcvCat {
		case MsgInit:
			// Traitement spécial pour vague d'init
			traiterInit(msg)
		case MsgPropagation:
			// Propagation descendante
			transmettreAuxEnfants(msg)
		case MsgType:
			// Autres types de message
			versControleur(msg)
		}
	}
}

