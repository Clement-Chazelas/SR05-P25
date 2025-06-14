package main

/*
Ce fichier implémente un algorithme permettant la capture de snapshot distribuée datée avec horloges vectorielles
Il permet de réaliser une capture cohérente de la blockchain, en sauvegardant les messages prépost.
*/

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type messageSnapType string

// Constantes correspondant aux types de message pour l'algorithme de snapshot
const (
	prepost messageSnapType = "pre"
	state   messageSnapType = "sta"
)

// Horloge vectorielle du controleur
var vectorClock []int = make([]int, NbSite)

// Indice du dernier controlleur ajouté
var newContIndex = -1
var quitContIndex = -1

var localSnapshot Snapshot       // Snapshot locale du controleur
var myColor string = "white"     // Couleur du controleur
var initiator bool = false       // Booléen indiquant si le site est initiateur
var nbLocalStateReceived int = 0 // Utilisé par l'initiateur, indique le nombre d'états locaux reçu

// Copie locale de la blockchain de l'application
var localBlockchain SerializableBlockchain

// Snapshot contient l'état des données locales lors de la capture
// Les attributs sont des tableaux pour que l'initiateur puisse agréger les différentes captures
// Les données du site i seront stockée à l'indice i des tableaux
type Snapshot struct {
	LocalState    []SerializableBlockchain `json:"Blockchains"` // Etat de la blockchain
	VectorClock   [][]int                  `json:"VectorClock"` // Horloge vectorielle au moment de la capture
	Timestamp     time.Time                `json:"timestamp"`
	ChannelStates [][]string               `json:"ChannelStates"` // Listes des messages prépost reçus par le site i
	//PendingTransaction [][]Transaction
}

// InitSnapshot réalise une snapshot locale et marque le site comme initiateur
func InitSnapshot() {
	display_d("Snapshot", "initialisation de la snapshot")
	myColor = "red"
	initiator = true

	// Sauvegarde locale, initialisation des tableaux pour stocker les états des autres sites
	localSnapshot = Snapshot{
		LocalState:    make([]SerializableBlockchain, NbSite),
		VectorClock:   make([][]int, NbSite),
		ChannelStates: make([][]string, NbSite),
		Timestamp:     time.Now(),
	}

	// Remplissage des tableaux à l'indice MyId avec la blockchain locale et l'horloge vectorielle
	localSnapshot.LocalState[MyId] = localBlockchain
	localSnapshot.VectorClock[MyId] = copyVectorClock(vectorClock)

}

// sendSnapshotMessage envoie un message de snapshot formaté (type + contenu)
func sendSnapshotMessage(msgType messageSnapType, data string) {

	formattedMsg := MsgFormat(MsgSender, Nom) +
		MsgFormat(MsgCategory, snapshot) +
		MsgFormat(MsgType, string(msgType)) +
		MsgFormat(MsgColor, myColor) + MsgFormat(MsgData, data)

	fmt.Println(formattedMsg)
}

// ReceiveAppMessage traite un message applicatif reçu d'un autre controleur et déclenche la capture si besoin.
// Gère égalemment la détection des messages prépost
func ReceiveAppMessage(msg string) {
	sdrClock := StrToClock(findval(msg, MsgHorloge))

	// Mise à jour de mon horloge vectorielle
	vectorClock = mergeVectorClocks(vectorClock, sdrClock)
	vectorClock[MyId]++

	c := findval(msg, MsgColor)
	if c == "red" && myColor == "white" {
		// L'expéditeur et rouge, et je n'ai pas encore réalisé ma capture d'instantané
		myColor = "red"

		// Création de la snapshot locale avec la blockchain et l'horloge vectorielle
		localSnapshot = Snapshot{
			LocalState:    []SerializableBlockchain{localBlockchain},
			VectorClock:   [][]int{copyVectorClock(vectorClock)},
			ChannelStates: make([][]string, NbSite),
			Timestamp:     time.Now(),
		}

		//Conversion de la snapshot locale en string et envoie à l'initiateur
		strSnap, _ := json.Marshal(localSnapshot)
		sendSnapshotMessage(state, string(strSnap))
	}

	if c == "white" && myColor == "red" {
		// J'ai réalisé ma capture, mais l'expéditeur ne l'a pas encore effectué
		// Il s'agit d'un message prépost

		if !initiator {
			// J'envoie un message de type prépost pour que l'initiateur puisse le conserver
			sendSnapshotMessage(prepost, msg)
		} else {
			// Je suis l'initiateur, je stocke directement le message prépost dans ma snapshot.
			localSnapshot.ChannelStates[MyId] = append(localSnapshot.ChannelStates[MyId], msg)
		}

	}
}

// ReceivePrepostMessage ajoute un message prépost au dans la liste du site concerné.
// Seul l'initiateur peut appeler cette fonction.
func ReceivePrepostMessage(msg string) {
	display_e("ReceivePrepostMessage", "Reception de ")
	rcvData := findval(msg, MsgData)

	// Récupération du nom de l'expéditeur pour en déduire son ID (indice dans liste Noms)
	sender := findval(msg, MsgSender)
	sdrId := sort.SearchStrings(Sites, sender)

	// Ajout du contenu du message à la snapshot de l'initiateur
	// Dans le tableau correspondant à l'ID de l'expéditeur
	localSnapshot.ChannelStates[sdrId] = append(localSnapshot.ChannelStates[sdrId], rcvData)

}

// ReceiveStateMessage ajoute un état local reçu à la snapshot de l'initiateur.
// Seul l'initiateur peut appeler cette fonction.
// Si l'initiateur à reçu tous les états locaux, il enregistre sa snapshot dans un fichier externe
func ReceiveStateMessage(msg string) {
	var rcvSnap Snapshot

	rcvData := findval(msg, MsgData)

	// Conversion des données reçues en variable Snapshot
	json.Unmarshal([]byte(rcvData), &rcvSnap)

	// Récupération du nom de l'expéditeur pour en déduire son ID (indice dans liste Noms)
	sender := findval(msg, MsgSender)
	sdrId := sort.SearchStrings(Sites, sender)
	if len(rcvSnap.LocalState) == 0 {
		stderr.Println(Nom, "Erreur, snapshot vide", msg)
	}
	// J'ajoute la blockchain reçue à ma snapshot locale (indice correspondant à l'ID)
	localSnapshot.LocalState[sdrId] = rcvSnap.LocalState[0]
	localSnapshot.VectorClock[sdrId] = rcvSnap.VectorClock[0]

	nbLocalStateReceived++

	// Vérifie si tous les états locaux ont été reçus
	if nbLocalStateReceived == NbSite-1 {

		//La snapshot est finie, on la stocke dans un fichier texte
		file, _ := os.OpenFile("sauvegarde.txt", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		defer file.Close()

		for i := 0; i < NbSite; i++ {
			file.WriteString("Blockchain site " + strconv.Itoa(i) + " : " + SendBlockchain(localSnapshot.LocalState[i].ToBlockchain()) + "\n\n")
			file.WriteString("Messages prépost pour le site " + strconv.Itoa(i) + " : " + strings.Join(localSnapshot.ChannelStates[i], "\n") + "\n\n")
			file.WriteString("Date sauvegarde pour le site " + strconv.Itoa(i) + " : " + ClockToStr(localSnapshot.VectorClock[i]) + "\n\n")
		}

		display_d("Snapshot", "Snapshot finie")
	}

}

// copyVectorClock copie une horloge vectorielle et retourne une nouvelle variable
func copyVectorClock(clock []int) []int {
	copyClock := make([]int, len(clock))
	copy(copyClock, clock)
	return copyClock
}

// mergeVectorClocks récupère le max de chaque ligne des deux horloges vectorielles.
// Incrémente la ligne correspondant au controleur.
func mergeVectorClocks(vc1, vc2 []int) []int {
	merged := make([]int, len(vc1))
	if len(vc1) > len(vc2) {
		stderr.Println(Nom, "l'horloge vectorielle 1 est plus grande que la 2", vc2)
		// Ce message provient d'un site qui n'a pas encore mis à jour son horloge vectorielle
		vc2 = addSiteToClock(vc2, newContIndex)
	} else if len(vc1) < len(vc2) {
		// Ce message provient d'un site qui n'a pas encore mis à jour son horloge vectorielle
		stderr.Println(Nom, "l'horloge vectorielle 1 est plus petite que la 2", vc1)
		vc2 = removeSiteFromClock(vc2, quitContIndex)
	}

	for i := range vc1 {
		if vc1[i] > vc2[i] {
			merged[i] = vc1[i]
		} else {
			merged[i] = vc2[i]
		}
	}

	return merged
}

func addSiteToClock(clock []int, index int) []int {
	newClock := copyVectorClock(clock)

	if index == len(clock) {
		newClock = append(newClock, 0)
	} else {
		newClock = append(newClock, 0)

		copy(newClock[index+1:], newClock[index:])
		newClock[index] = 0
	}
	return newClock

}

func removeSiteFromClock(clock []int, index int) []int {
	newClock := copyVectorClock(clock)
	newClock = append(newClock[:index], newClock[index+1:]...)
	return newClock
}

// ClockToStr convertit une horloge vectorielle en string (ex: "1,0,2")
func ClockToStr(clock []int) string {
	strs := make([]string, len(clock))
	for i, v := range clock {
		strs[i] = strconv.Itoa(v)
	}
	return strings.Join(strs, ",")
}

// StrToClock convertit un string en d’horloge vectorielle
func StrToClock(clockStr string) []int {
	parts := strings.Split(clockStr, ",")
	clock := make([]int, len(parts))
	for i, val := range parts {
		clock[i], _ = strconv.Atoi(val)
	}
	return clock
}
