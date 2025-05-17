package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type messageSnapType string

const (
	prepost messageSnapType = "pre"
	state   messageSnapType = "sta"
)

// HORLOGE VECTORIELLE
var vectorClock []int = make([]int, NbSite)

// VARIABLES
var localSnapshot Snapshot
var myColor string = "white"
var initiator bool = false

var localBlockchain SerializableBlockchain

// SNAPSHOT
type Snapshot struct {
	LocalState    []SerializableBlockchain `json:"Blockchains"` // A voir comment on récupère la blockchain
	VectorClock   []int                    `json:"VectorClock"` // Pour savoir quand a été prise la snapshot
	Timestamp     time.Time                `json:"timestamp"`
	ChannelStates [][]string               `json:"ChannelStates"` // Messages prepost
	//PendingTransaction [][]Transaction
}

// Utilitaires
func copyVectorClock(clock []int) []int {
	copyClock := make([]int, len(clock))
	copy(copyClock, clock)
	return copyClock
}

func mergeVectorClocks(vc1, vc2 []int) []int {
	merged := make([]int, len(vc1))
	for i := range vc1 {
		if vc1[i] > vc2[i] {
			merged[i] = vc1[i]
		} else {
			merged[i] = vc2[i]
		}
	}
	return merged
}

// Fonction de début, initialisation de la snapshot par un site
func InitSnapshot() {
	display_d("Snapshot", "initialisation de la snapshot")
	myColor = "red"
	initiator = true

	// Sauvegarde locale
	localSnapshot = Snapshot{
		LocalState:    make([]SerializableBlockchain, NbSite),
		VectorClock:   copyVectorClock(vectorClock),
		ChannelStates: make([][]string, NbSite),
		Timestamp:     time.Now(),
	}
	localSnapshot.LocalState[MyId] = localBlockchain

}

// Fonction d'envoi des messages
func sendSnapshotMessage(msgType messageSnapType, data string) {

	formattedMsg := MsgFormat(MsgSender, Nom) +
		MsgFormat(MsgCategory, snapshot) +
		MsgFormat(MsgType, string(msgType)) +
		MsgFormat(MsgColor, myColor) + MsgFormat(MsgData, data)

	fmt.Println(formattedMsg)
}

// Fonction pour la réception d'un message applicatif
func ReceiveAppMessage(msg string) {
	sdrClock := StrToClock(findval(msg, MsgHorloge))
	vectorClock = mergeVectorClocks(vectorClock, sdrClock)

	c := findval(msg, MsgColor)
	if c == "red" && myColor == "white" {
		myColor = "red"

		localSnapshot = Snapshot{
			LocalState:    []SerializableBlockchain{localBlockchain}, // à adapter
			VectorClock:   copyVectorClock(vectorClock),
			ChannelStates: make([][]string, NbSite),
			Timestamp:     time.Now(),
		}

		//Conversion de la snapshot locale en string et envoie à l'initiateur
		strSnap, _ := json.Marshal(localSnapshot)
		sendSnapshotMessage(state, string(strSnap))
	}

	if c == "white" && myColor == "red" {
		sendSnapshotMessage(prepost, msg)
	}
}

func ReceivePrepostMessage(msg string) {

	rcvData := findval(msg, MsgData)

	sender := findval(msg, MsgSender)
	sdrId := sort.SearchStrings(Sites, sender)

	localSnapshot.ChannelStates[sdrId] = append(localSnapshot.ChannelStates[sdrId], rcvData)

}

func ReceiveStateMessage(msg string) {
	var rcvSnap Snapshot

	rcvData := findval(msg, MsgData)

	json.Unmarshal([]byte(rcvData), &rcvSnap)

	sender := findval(msg, MsgSender)
	sdrId := sort.SearchStrings(Sites, sender)
	// J'ajoute la blockchain reçue à ma snapshot locale
	localSnapshot.LocalState[sdrId] = rcvSnap.LocalState[0]
	//Peut etre Transaction

	// Demander au prof quoi faire avec l'horloge

}

func ClockToStr(clock []int) string {
	strs := make([]string, len(clock))
	for i, v := range clock {
		strs[i] = strconv.Itoa(v)
	}
	return strings.Join(strs, ",")
}

func StrToClock(clockStr string) []int {
	parts := strings.Split(clockStr, ",")
	clock := make([]int, len(parts))
	for i, val := range parts {
		clock[i], _ = strconv.Atoi(val)
	}
	return clock
}

/*

Fonction strToCLock et ClockToStr
Remplacer updateVectorClock par mergeVectorClock



Importer le fichier blockchainStruct et serializeStruct dans ce projet et ajouter blockchainToStr et strToBlockchain
Modifier la structure Snapshot pour qu'elle contienne une blockchain
Faire les fonctions SnapshotToStr et StrToSnapshot en utilisant Json



faire un schéma de l'échange des messages entre les sites


Faire le script démarrer snapshot

Coté application, attendre la fin des go routines

Faire le print de la snapshot dans un fichier

Si le temps, gérer l'arret de la blockchain


Faire que l'app envoie sa blockchain au controleur

Canva : Présentation blockchain générale
Présentation intégration dans notre application avec file d'attente
Présentation Snapshot


*/
