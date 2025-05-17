package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type messageSnapType string

const (
	prepost messageSnapType = "pre"
	state messageSnapType = "sta"
)

// HORLOGE VECTORIELLE
var vectorClock []int = make([]int, NbSite)

// VARIABLES
var localSnapshot *Snapshot = nil
var color string = "white"
var initiator bool = false

// SNAPSHOT
type Snapshot struct {
	LocalState 		string // A voir comment on récupère la blockchain
	VectorClock   []int // Pour savoir quand a été prise la snapshot
	Timestamp     time.Time
	ChannelStates [][]string // Messages dans canal pendant prepost
	//PendingTransaction [][]Transaction
}

// Utilitaires
func copyVectorClock(clock []int) []int {
	copyClock := make([]int, len(clock))
	copy(copyClock, clock)
	return copyClock
}

func UpdateVectorClock(msg string) {
	rcvClockStr := findval(msg, MsgHorloge)
	if rcvClockStr != "" {
		parts := strings.Split(rcvClockStr, ",")
		for i, val := range parts {
			intVal, _ := strconv.Atoi(val)
			if intVal > vectorClock[i] {
				vectorClock[i] = intVal
			}
		}
	}
	vectorClock[MyId]++
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
	color = "red"
	initiator = true

	// Sauvegarde locale
	snapshotData = &Snapshot{
		LocalState:    "état blockchain courant initial", // à adapter
		VectorClock:   copyVectorClock(vectorClock),
		ChannelStates: make([][]string, NbSite),
		Timestamp:     time.Now(),
	}

	sendSnapshotMessage(prepost)
}

// Fonction d'envoi des messages
func sendSnapshotMessage(msgType messageSnapType) {
	vectorClock[MyId]++

	strs := make([]string, len(vectorClock))
	for i, v := range vectorClock {
		strs[i] = strconv.Itoa(v)
	}
	vcString := strings.Join(strs, ",")

	formattedMsg := MsgFormat(MsgSender, Nom) +
		MsgFormat(MsgCategory, snapshot) +
		MsgFormat(MsgType, string(msgType)) +
		MsgFormat(MsgHorloge, vcString) +
		MsgFormat(MsgColor, color)
		
	fmt.Println(formattedMsg)
}


// Fonction pour la réception d'un message applicatif
func ReceiveAppMessage(msg string){
	UpdateVectorClock(msg)

	c := findval(msg, MsgColor)
	if c == "red" && color == "white" {
		color = "red"

		localSnapshot = &Snapshot{
			LocalState:    "état blockchain courant au déclenchement", // à adapter
			VectorClock:   copyVectorClock(vectorClock),
			ChannelStates: make([][]string, NbSite),
			Timestamp:     time.Now(),
		}
		sendSnapshotMessage(state)
	}

	if c == "white" && color == "red" {
		sendSnapshotMessage(prepost)
	}
}

func ReceivePrepostMessage(msg string) {
	UpdateVectorClock(msg)

	rcvData := findval(msg, MsgData)

	if initiator {
		sender := findval(msg, MsgSender)
		sdrId := sort.SearchStrings(Sites, sender)
		if localSnapshot != nil {
			localSnapshot.ChannelStates[sdrId] = append(localSnapshot.ChannelStates[sdrId], rcvData)
		}
	} else {
		sendSnapshotMessage(prepost)
	}
}

func ReceiveStateMessage(msg string) {
	UpdateVectorClock(msg)

	if initiator {
		localSnapshot.VectorClock = mergeVectorClocks(localSnapshot.VectorClock, vectorClock)
	} else {
		sendSnapshotMessage(state)
	}
}
