package main

/*
Ce fichier implémente l'algorithme de la file d'attente répartie avec estampilles.
Il gère la coordination entre les applications pour l’accès exclusif à la section critique (écriture sur la blockchain)
*/

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

type messageFileType string

// Constantes correspondant aux types de message pour la file d’attente
const (
	request messageFileType = "req"
	release messageFileType = "rel"
	ack     messageFileType = "ack"
)

// messageFile correspond à un message utilisé par l'algo de la file d'attente répartie
// Il possède un type et une date (estampille)
type messageFile struct {
	Type messageFileType `json:"type"` // type du message
	Date int             `json:"date"` // estampille
}

// Chaque site est identifié par son ID (index)
// La file d'attente est un tableau de taille nbSite
// La file d’attente (fileAtt) contient le dernier message reçu (à indice i) de chaque site (i)
var fileAtt = make([]messageFile, NbSite)

// Estampille du controleur
var estamp int = 0

// Estampille de ma dernière requête autorisée
// Utilisée pour éviter d'accorder deux fois la même requête
var estampLastReq int = 0

// receiveDemandeSC traite une demande de l'application d’entrée en section critique
func receiveDemandeSC() {
	estamp++

	// Création du message de type requête et ajout dans ma file d'attente
	newMessage := messageFile{Type: request, Date: estamp}
	fileAtt[MyId] = newMessage

	// Envoi de la requête aux autres controleurs
	sendFileMessage(request)
}

// receiveFinSC traite la sortie de la section critique de l'application
func receiveFinSC() {
	estamp++

	// Création du message de type release et ajout dans ma file d'attente
	newMessage := messageFile{Type: release, Date: estamp}
	fileAtt[MyId] = newMessage

	// Envoi de la libération aux autres controleurs
	sendFileMessage(release)
}

// receiveRequest traite une requête en provenance d'un autre controleur.
// Elle met à jour l’estampille locale et sa file d'attente
// Envoie un ack et vérifie si le controleur peut entrer en SC
func receiveRequest(j, h int) {

	// Maj de l'estampille et ajout de la requête dans la file d'attente
	estamp = maxInt(estamp, h) + 1
	fileAtt[j] = messageFile{Type: request, Date: h}

	// Envoi de l'ack
	sendFileMessage(ack)

	// Vérifie si j'ai une requête et que son estampille est la plus petite
	if fileAtt[MyId].Type == request && isOldestRequest() {
		//Envoi DébutSC à l'Application
		fmt.Printf("CONT:debutSC\n")
	}

}

// receiveRelease traite une libération en provenance d'un autre controleur.
// Elle enregistre la libération de la SC par un autre site
// et vérifie si le controleur peut entrer en SC
func receiveRelease(j, h int) {

	// Maj de l'estampille et ajout de la libération dans la file d'attente
	estamp = maxInt(estamp, h) + 1
	fileAtt[j] = messageFile{Type: release, Date: h}

	// Vérifie si j'ai une requête et que son estampille est la plus petite
	if fileAtt[MyId].Type == request && isOldestRequest() {
		//Envoi DébutSC à l'Application
		fmt.Printf("CONT:debutSC\n")
	}
}

// receiveAck traite un ack en provenance d'un autre controleur.
// Elle enregistre l'ack si nécessaire et vérifie si le controleur peut entrer en SC
func receiveAck(j, h int) {

	// Maj de l'estampille
	estamp = maxInt(estamp, h) + 1

	// Un ack ne peut pas remplacer un message de type requête
	if fileAtt[j].Type != request {
		fileAtt[j] = messageFile{Type: ack, Date: h}
	}

	// Vérifie si j'ai une requête et que son estampille est la plus petite
	if fileAtt[MyId].Type == request && isOldestRequest() {
		//Envoi DébutSC à l'Application
		fmt.Printf("CONT:debutSC\n")
	}

}

// sendFileMessage envoie un message de type req/rel/ack aux autres controleur
func sendFileMessage(msgType messageFileType) {
	// Formatage du message incluant l'estampille et le type
	newMessage := MsgFormat(MsgSender, Nom) + MsgFormat(MsgCategory, file) + MsgFormat(MsgEstampille, strconv.Itoa(estamp)) +
		MsgFormat(MsgType, string(msgType))

	// Envoi du message
	fmt.Println(newMessage)
}

// ReceiveFileMessage est appelée à la réception d'un message de catégorie file.
// Elle traite le message reçu selon son type (req, rel, ack).
func ReceiveFileMessage(msg string) {

	rcvType := messageFileType(findval(msg, MsgType))
	sdrEstamp, _ := strconv.Atoi(findval(msg, MsgEstampille))

	// Récupération du nom de l'expéditeur pour en déduire son ID (indice dans liste Noms)
	sender := findval(msg, MsgSender)
	sdrId := sort.SearchStrings(Sites, sender)

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

// ReceiveSC traite une demande reçue de l'application pour entrer ou sortir de la section critique
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

// infCouple compare deux couples d'entier pour définir le plus petit
func infCouple(a, b [2]int) bool {
	if a[0] < b[0] {
		return true
	}
	if a[0] == b[0] && a[1] < b[1] {
		return true
	}
	return false
}

// isOldestRequest détermine si le site courant à la plus ancienne requête en attente
func isOldestRequest() bool {

	// L'estampille de ma requête correspond à la dernière requête acceptée
	if fileAtt[MyId].Date == estampLastReq {
		// Requête déjà traitée
		return false
	}

	// Parcours de la file d'attente
	for id, msg := range fileAtt {

		if id == MyId {
			// Je ne traite pas ma propre requête
			continue
		}

		// Vérifie que le couple (date, id) est le plus petit de tous et que la file d'attente n'a pas de case vide
		if !infCouple([2]int{fileAtt[MyId].Date, MyId}, [2]int{msg.Date, id}) || msg.Date == 0 {
			return false
		}
	}

	// La requête devient la dernière acceptée
	estampLastReq = fileAtt[MyId].Date
	return true
}

// addSiteToFile permet d'ajouter une case à la file d'attente à l'index précisé
func addSiteToFile(newSiteIndex int) []messageFile {
	newFile := make([]messageFile, NbSite)
	copy(newFile, fileAtt)
	// Si l'indice est à la fin de la slice, nous pouvons simplement retourner la file.
	if newSiteIndex != len(fileAtt) {
		// Sinon, il faut décaler les cases suivantes
		copy(newFile[newSiteIndex+1:], newFile[newSiteIndex:])

		newFile[newSiteIndex] = messageFile{}
	}

	return newFile
}

// removeSiteFromFile permet de supprimer une case à la file d'attente à l'index précisé
func removeSiteFromFile(siteIndex int) []messageFile {
	newFile := make([]messageFile, NbSite)
	newFile = append(fileAtt[:siteIndex], fileAtt[siteIndex+1:]...)
	return newFile
}

// sendFileAtt permet de transformer une file d'attente en string pour pouvoir l'envoyer en message.
func sendFileAtt(msg []messageFile) string {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

// receiveFileAtt permet de récuperer une file d'attente d'un string à la réception d'un message.
func receiveFileAtt(jsonString string) []messageFile {
	var msg []messageFile
	err := json.Unmarshal([]byte(jsonString), &msg)
	if err != nil {
		return []messageFile{}
	}
	return msg
}
