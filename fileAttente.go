package main

import (
	"fmt"
	"sort"
	"strconv"
)

type messageFileType string

const (
	request messageFileType = "req"
	release messageFileType = "rel"
	ack     messageFileType = "ack"
)

type messageFile struct {
	Type messageFileType
	Date int
}

// La file d'attente est un tableau de taille nbSite
// l'indice i du tableau correspond au dernier msg reçu du site i
// un message est un type et une date
var fileAtt = make([]messageFile, NbSite)

var estamp int = 0
var estampLastReq int = 0

func receiveDemandeSC() {
	estamp++
	newMessage := messageFile{Type: request, Date: estamp}
	fileAtt[MyId] = newMessage
	//display_d("receiveDemandeSC", "Envoie requête")
	sendFileMessage(request)
}

func receiveFinSC() {
	estamp++
	newMessage := messageFile{Type: release, Date: estamp}
	fileAtt[MyId] = newMessage
	//display_d("receiveFinSC", "Envoie Release")
	sendFileMessage(release)
}

func receiveRequest(j, h int) {
	estamp = maxInt(estamp, h) + 1
	fileAtt[j] = messageFile{Type: request, Date: h}
	//display_d("receiveRequest", "Envoie ACK")
	sendFileMessage(ack)

	if fileAtt[MyId].Type == request && isOldestRequest() {
		//Envoyer DébutSC à l'App
		fmt.Printf("CONT:debutSC\n")
	}

}

func receiveRelease(j, h int) {
	estamp = maxInt(estamp, h) + 1
	fileAtt[j] = messageFile{Type: release, Date: h}

	if fileAtt[MyId].Type == request && isOldestRequest() {
		//Envoyer DébutSC à l'App
		fmt.Printf("CONT:debutSC\n")
	}
}

func receiveAck(j, h int) {
	estamp = maxInt(estamp, h) + 1
	if fileAtt[j].Type != request {
		fileAtt[j] = messageFile{Type: ack, Date: h}
	}
	//display_d("receiveAck", "Ack Reçu")
	if fileAtt[MyId].Type == request && isOldestRequest() {
		//Envoyer DébutSC à l'App
		fmt.Printf("CONT:debutSC\n")
	}

}

func sendFileMessage(msgType messageFileType) {
	newMessage := MsgFormat(MsgSender, Nom) + MsgFormat(MsgCategory, file) + MsgFormat(MsgEstampille, strconv.Itoa(estamp)) +
		MsgFormat(MsgType, string(msgType))
	fmt.Println(newMessage)
}

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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func infCouple(a, b [2]int) bool {
	if a[0] < b[0] {
		return true
	}
	if a[0] == b[0] && a[1] < b[1] {
		return true
	}
	return false
}

func isOldestRequest() bool {
	//tderr.Println(Nom, MyId, fileAtt)

	if fileAtt[MyId].Date == estampLastReq {
		// Requête déjà traitée
		return false
	}

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
