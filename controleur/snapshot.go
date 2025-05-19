// Gère le protocole de snapshot distribué avec horloges vectorielles
// Permet de figer l’état de la blockchain et des messages prépost entre les sites
// Utilisé uniquement côté contrôleur

package main


import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Déclare les types de message pour le protocole de snapshot
type messageSnapType string

const (
	prepost messageSnapType = "pre" // Message "pre" contenant un message prépost
	state   messageSnapType = "sta" // Message "state" contenant un état local
)

// HORLOGE VECTORIELLE - partagée entre tous les sites
var vectorClock []int = make([]int, NbSite)

// VARIABLES de SNAPSHOT
var localSnapshot Snapshot   // Snapshot locale en cours
var myColor string = "white" // État du site : "white" = pas encore participé, "red" = snapshot déclenché
var initiator bool = false 	 // Indique si le site a initié le snapshot

var localBlockchain SerializableBlockchain  //Blockchain locale sérialisée

// Snapshot structure les données capturées localement lors du snapshot
type Snapshot struct {
	LocalState    []SerializableBlockchain `json:"Blockchains"`   // Blockchain locale de chaque site
	VectorClock   []int                    `json:"VectorClock"`   // Horloge vectorielle au moment du snapshot
	Timestamp     time.Time                `json:"timestamp"`	  // Date et heure de capture
	ChannelStates [][]string               `json:"ChannelStates"` // Messages prepost
}

// copyVectorClock copie une horloge vectorielle dans un nouveau tableau
func copyVectorClock(clock []int) []int {
	copyClock := make([]int, len(clock))
	copy(copyClock, clock)
	return copyClock
}

// mergeVectorClocks fusionne deux horloges vectorielles composante par composante (max)
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

// InitSnapshot initialise une snapshot locale et marque le site comme initiateur
func InitSnapshot() {
	display_d("Snapshot", "initialisation de la snapshot")
	myColor = "red"
	initiator = true

	// Capture l’état local : blockchain + horloge
	localSnapshot = Snapshot{
		LocalState:    make([]SerializableBlockchain, NbSite),
		VectorClock:   copyVectorClock(vectorClock),
		ChannelStates: make([][]string, NbSite),
		Timestamp:     time.Now(),
	}
	localSnapshot.LocalState[MyId] = localBlockchain

}

// sendSnapshotMessage envoie un message de snapshot formaté (type + contenu)
func sendSnapshotMessage(msgType messageSnapType, data string) {

	formattedMsg := MsgFormat(MsgSender, Nom) +
		MsgFormat(MsgCategory, snapshot) +
		MsgFormat(MsgType, string(msgType)) +
		MsgFormat(MsgColor, myColor) + MsgFormat(MsgData, data)

	fmt.Println(formattedMsg)
}

// ReceiveAppMessage traite un message applicatif reçu et déclenche un snapshot si besoin
func ReceiveAppMessage(msg string) {
	sdrClock := StrToClock(findval(msg, MsgHorloge))
	vectorClock = mergeVectorClocks(vectorClock, sdrClock)

	c := findval(msg, MsgColor)

	// Déclenche le snapshot si message rouge reçu pour la première fois
	if c == "red" && myColor == "white" {
		myColor = "red"

		localSnapshot = Snapshot{
			LocalState:    []SerializableBlockchain{localBlockchain}, // à adapter
			VectorClock:   copyVectorClock(vectorClock),
			ChannelStates: make([][]string, NbSite),
			Timestamp:     time.Now(),
		}

		// Sérialise la snapshot et envoie l’état local au site initiateur
		strSnap, _ := json.Marshal(localSnapshot)
		sendSnapshotMessage(state, string(strSnap))
	}

	// Envoie les messages reçus comme "prepost" s’il est déjà rouge
	if c == "white" && myColor == "red" {
		sendSnapshotMessage(prepost, msg)
	}
}

// ReceivePrepostMessage ajoute un message prépost au canal du site concerné
func ReceivePrepostMessage(msg string) {

	rcvData := findval(msg, MsgData)

	sender := findval(msg, MsgSender)
	sdrId := sort.SearchStrings(Sites, sender)

	localSnapshot.ChannelStates[sdrId] = append(localSnapshot.ChannelStates[sdrId], rcvData)

}

// ReceiveStateMessage ajoute un état reçu (EGi) à la snapshot locale (depuis un autre site)
func ReceiveStateMessage(msg string) {
	var rcvSnap Snapshot

	rcvData := findval(msg, MsgData)

	json.Unmarshal([]byte(rcvData), &rcvSnap)

	sender := findval(msg, MsgSender)
	sdrId := sort.SearchStrings(Sites, sender)

	// Stocke la blockchain du site distant
	localSnapshot.LocalState[sdrId] = rcvSnap.LocalState[0]

	// Demander au prof quoi faire avec l'horloge

}

// ClockToStr convertit une horloge vectorielle en chaîne (ex: "1,0,2")
func ClockToStr(clock []int) string {
	strs := make([]string, len(clock))
	for i, v := range clock {
		strs[i] = strconv.Itoa(v)
	}
	return strings.Join(strs, ",")
}

// StrToClock convertit une chaîne d’horloge vectorielle en tableau d’int
func StrToClock(clockStr string) []int {
	parts := strings.Split(clockStr, ",")
	clock := make([]int, len(parts))
	for i, val := range parts {
		clock[i], _ = strconv.Atoi(val)
	}
	return clock
}
