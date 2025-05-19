// Implémente une file d'attente répartie avec estampilles logiques
// Gère la coordination entre sites pour l’accès exclusif à la section critique
// Chaque site peut demander, libérer et accorder un droit d’accès à la SC en respectant l’ordre logique

package main

import (
	"fmt"
	"sort"
	"strconv"
)

// Déclare les types de message pour la file d’attente
type messageFileType string

const (
	request messageFileType = "req" // Demande d’entrée en section critique
	release messageFileType = "rel" // Libération de la section critique
	ack     messageFileType = "ack" // Accusé de réception
)

// messageFile stocke un message avec son type et une date
type messageFile struct {
	Type messageFileType
	Date int
}

// Chaque site est identifié par son ID (index)
// La file d'attente est un tableau de taille nbSite
// La file d’attente (fileAtt) contient le dernier message reçu (indice i) de chaque site (i)
// un message est un type et une date
var fileAtt = make([]messageFile, NbSite)

var estamp int = 0         // Estampille logique locale
var estampLastReq int = 0  // Estampille de la dernière requête traitée

// receiveDemandeSC traite une demande locale d’entrée en section critique
func receiveDemandeSC() {
	estamp++
	newMessage := messageFile{Type: request, Date: estamp}
	fileAtt[MyId] = newMessage
	sendFileMessage(request)
}

// receiveFinSC traite la sortie de la section critique locale
func receiveFinSC() {
	estamp++
	newMessage := messageFile{Type: release, Date: estamp}
	fileAtt[MyId] = newMessage
	sendFileMessage(release)
}

// receiveRequest met à jour l’estampille locale et enregistre une demande distante
// Envoie un ack et vérifie si le site courant peut entrer en SC
func receiveRequest(j, h int) {
	estamp = maxInt(estamp, h) + 1
	fileAtt[j] = messageFile{Type: request, Date: h}
	sendFileMessage(ack)

	if fileAtt[MyId].Type == request && isOldestRequest() {
		//Envoyer DébutSC à l'App
		fmt.Printf("CONT:debutSC\n")
	}

}

// receiveRelease enregistre la libération de SC par un autre site et vérifie si le site courant est éligible
func receiveRelease(j, h int) {
	estamp = maxInt(estamp, h) + 1
	fileAtt[j] = messageFile{Type: release, Date: h}

	if fileAtt[MyId].Type == request && isOldestRequest() {
		//Envoyer DébutSC à l'App
		fmt.Printf("CONT:debutSC\n")
	}
}

// receiveAck traite la réception d’un accusé (ack) et vérifie l’éligibilité à la SC
func receiveAck(j, h int) {
	estamp = maxInt(estamp, h) + 1
	if fileAtt[j].Type != request {
		fileAtt[j] = messageFile{Type: ack, Date: h}
	}
	if fileAtt[MyId].Type == request && isOldestRequest() {
		//Envoyer DébutSC à l'App
		fmt.Printf("CONT:debutSC\n")
	}

}

// sendFileMessage envoie un message de type req/rel/ack aux autres sites via le contrôleur
func sendFileMessage(msgType messageFileType) {
	newMessage := MsgFormat(MsgSender, Nom) + MsgFormat(MsgCategory, file) + MsgFormat(MsgEstampille, strconv.Itoa(estamp)) +
		MsgFormat(MsgType, string(msgType))
	fmt.Println(newMessage)
}

// ReceiveFileMessage redirige un message reçu selon son type (req, rel, ack)
func ReceiveFileMessage(msg string) {
	rcvType := messageFileType(findval(msg, MsgType))
	sender := findval(msg, MsgSender)
	sdrId := sort.SearchStrings(Sites, sender)
	sdrEstamp, _ := strconv.Atoi(findval(msg, MsgEstampille))
	switch rcvType {
	case request:
		receiveRequest(sdrId, sdrEstamp)
		break
	case release:
		receiveRelease(sdrId, sdrEstamp)
		break
	case ack:
		receiveAck(sdrId, sdrEstamp)
		break
	}

}

// ReceiveSC traite une commande locale reçue de l’application (demandeSC ou finSC)
func ReceiveSC(msg string) {
	switch msg {
	case "demandeSC":
		receiveDemandeSC()
		break
	case "finSC":
		receiveFinSC()
		break
	}
}

// maxInt retourne le maximum entre deux entiers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// infCouple compare deux couples (estampille, id) pour définir une priorité
func infCouple(a, b [2]int) bool {
	if a[0] < b[0] {
		return true
	}
	if a[0] == b[0] && a[1] < b[1] {
		return true
	}
	return false
}

// isOldestRequest détermine si le site courant a la plus ancienne requête en attente
func isOldestRequest() bool {
	//tderr.Println(Nom, MyId, fileAtt)

	if fileAtt[MyId].Date == estampLastReq {
		// Requête déjà traitée
		return false
	} // Vérifie si la requête courante est déjà traitée

	for id, msg := range fileAtt {
		if id == MyId {
			// Je ne traite pas ma propre requête
			continue
		}

		// Vérif que le couple (date, id) est le plus petit de tous et qu'on a bien reçu les ACK (ou autre) de tt le monde (date !=0)
		if !infCouple([2]int{fileAtt[MyId].Date, MyId}, [2]int{msg.Date, id}) || msg.Date == 0 {
			return false
		}
	}
	estampLastReq = fileAtt[MyId].Date
	return true
}
