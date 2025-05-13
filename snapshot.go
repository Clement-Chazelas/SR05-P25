package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"
)

var l = log.New(os.Stderr, "", log.LstdFlags)

// HORLOGE VECTORIELLE

type VectorClock map[int]int

func InitiateVectorClock(nbSites int) VectorClock {
	vc := make(VectorClock)
	for i := 0; i < nbSites; i++ {
		vc[i] = 0
	}
	return vc
}

func (vc VectorClock) Increment(siteID int) {
	vc[siteID]++
}

func (vc VectorClock) Update(otherVC VectorClock) {
	for id, time := range otherVC {
		if _, ok := vc[id]; !ok || vc[id] < time {
			vc[id] = time
		}
	}
	vc.Increment(MyId)
}

func copyVectorClock(vc VectorClock) VectorClock {
	newVC := make(VectorClock)
	for id, time := range vc {
		newVC[id] = time
	}
	return newVC
}

func getCurrentVectorClock() VectorClock {
	vc := make(VectorClock)

	for i := 0; i < NbSite; i++ {
		vc[i] = 0 // On initialise les autres sites avec zéro
	}

	return vc
}

// SNAPSHOT
type Snapshot struct {
	Blockchain    Blockchain
	UTXOSet       UTXOSet
	VectorClock   VectorClock
	Timestamp     time.Time
	ChannelStates map[int][]string
}

// Fonction pour enregistrer les snapshots
// func (s Snapshot) Save(filename string) error {
// 	// sauvegarde snapshot
// }

// VARIABLES
const markerMessageType = "marker"

var hasTakenLocalSnapshot = false
var receivedMarkers = make(map[int]bool)
var channelStates = make(map[int][]string)
var snapshotState Snapshot
var snapshotVectorClock VectorClock
var snapshotMutex sync.Mutex

// FONCTIONS POUR ALGO
// Fonction pour initialiser l'instantané
func InitiateDistributedSnapshot(b Blockchain) {
	snapshotMutex.Lock()
	defer snapshotMutex.Unlock()

	fmt.Println(Nom, ": Initiation de l'instantané réparti")
	hasTakenLocalSnapshot = true

	snapshotState = TakeLocalSnapshot(b, b.GetLastBlock().UTXOs, getCurrentVectorClock())
	snapshotVectorClock = snapshotState.VectorClock

	// Envoi des marqueurs
	snapshotState.ChannelStates = make(map[int][]string)
	for i := 0; i < NbSite; i++ {
		if i != MyId {
			snapshotState.ChannelStates[i] = []string{}
			sendSnapshotMessage(Sites[i], markerMessageType, nil)
		}
	}
	fmt.Println(Nom, ": État local enregistré, marqueurs envoyés.")
}

// Fonction qui gère la réception d'un marqueur
func ReceiveSnapshotMessage(senderName string, messageType string, data string) {
	senderID := sort.SearchStrings(Sites, senderName)

	snapshotMutex.Lock()
	defer snapshotMutex.Unlock()

	if messageType == markerMessageType {
		fmt.Println(Nom, ": Marqueur reçu de", senderName)

		// Si pas encore instantané local
		if !hasTakenLocalSnapshot {
			fmt.Println(Nom, ": Premier marqueur reçu. Enregistrement de l'état local.")
			hasTakenLocalSnapshot = true
			snapshotState = TakeLocalSnapshot(blockchain, blockchain.GetLastBlock().UTXOs, getCurrentVectorClock())
			snapshotVectorClock = snapshotState.VectorClock

			// Initialisation des états des canaux pour les sites
			snapshotState.ChannelStates = make(map[int][]string)
			for i := 0; i < NbSite; i++ {
				if i != MyId {
					snapshotState.ChannelStates[i] = []string{}
					sendSnapshotMessage(Sites[i], markerMessageType, nil)
				}
			}
		}

		receivedMarkers[senderID] = true // Le site a envoyé un marqueur
		snapshotState.ChannelStates[senderID] = channelStates[senderID] // Sauvegarde les états des canaux du site
		channelStates[senderID] = []string{} // Réinitialise les canaux du site

	} else {
		if hasTakenLocalSnapshot && !receivedMarkers[senderID] {
			fmt.Println(Nom, ": Message application '", data, "' reçu de", senderName, "pendant l'instantané.")
			channelStates[senderID] = append(channelStates[senderID], data)
		}
	}

	// Vérification si tous les sites ont leur marqueur
	allMarkersReceived := true
	for i := 0; i < NbSite; i++ {
		if i != MyId && !receivedMarkers[i] {
			allMarkersReceived = false
			break
		}
	}

	// Si l'instantané local a été pris et que tous les marqueurs ont été reçus
	if hasTakenLocalSnapshot && allMarkersReceived {
		fmt.Println(Nom, ": Instantané global terminé.")
		saveGlobalSnapshot()
		resetSnapshotState()
	}
}

func resetSnapshotState() {
	hasTakenLocalSnapshot = false
	receivedMarkers = make(map[int]bool)
	channelStates = make(map[int][]string)
	snapshotState = Snapshot{}
	snapshotVectorClock = nil
}

// Fonction qui enregistre l'état global
func saveGlobalSnapshot() {
	filename := fmt.Sprintf("snapshot-%s-%s.gob", Nom, time.Now().Format("20060102150405"))
	snapshotState.VectorClock = snapshotVectorClock
	// Enregistrer le snapshot
	// err := snapshotState.Save(filename)
	// if err != nil {
	// 	fmt.Println(Nom, ": Erreur lors de la sauvegarde de l'instantané global:", err)
	// }
}

// Fonction d'envoi de message
func sendSnapshotMessage(receiverName string, messageType string, data interface{}) {
	msg := MsgFormat(MsgSender, Nom) +
		MsgFormat(MsgCategory, snapshot) +
		MsgFormat(MsgType, messageType)
	if data != nil {
		msg += MsgFormat(MsgData, fmt.Sprintf("%v", data))
	}
	fmt.Println(receiverName, msg)
}

// Fonction qui est appelée dans le contrôleur
func HandleSnapshotMessage(msg string) {
	sender := findval(msg, MsgSender)
	msgType := findval(msg, MsgType)
	data := findval(msg, MsgData)
	ReceiveSnapshotMessage(sender, msgType, data)
}

// Fonction de copie de l'état local
func TakeLocalSnapshot(b Blockchain, u UTXOSet, vc VectorClock) Snapshot {
	return Snapshot{
		Blockchain:    b, // ou copie
		UTXOSet:       u,
		VectorClock:   copyVectorClock(vc),
		Timestamp:     time.Now(),
		ChannelStates: make(map[int][]string),
	}
}
