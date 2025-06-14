package main

import (
	"math"
	"slices"
	"strconv"
)

var (
	parentTmp = 0
)

func DemarrerElection() {
	if parentTmp == 0 {
		stderr.Println(blanc, "["+Nom+"]", "Je démarre une election", raz)
		elu = MyId
		parentTmp = MyId
		nbVoisinsAttendus = NbVoisins
		envoyerA(election, bleu, strconv.Itoa(elu), ListVoisins)
	}
}

func RecevoirMessageBleu(msg string) {
	k, _ := strconv.Atoi(findval(msg, MsgData))
	senderId, _ := strconv.Atoi(findval(msg, MsgSender))
	if k < elu {
		stderr.Println(noir, "["+Nom+"]", "Je change de vague pour", k, raz)
		elu = k
		parentTmp = senderId
		nbVoisinsAttendus = NbVoisins - 1
		if nbVoisinsAttendus > 0 {
			envoyerAuxVoisinsSauf(election, bleu, strconv.Itoa(elu), senderId)
		} else {
			// j'envoie rouge à mon parent
			destinataire := []int{parentTmp}
			envoyerA(election, rouge, strconv.Itoa(elu), destinataire)
		}
	} else if elu == k {
		destinataire := []int{senderId}
		envoyerA(election, rouge, strconv.Itoa(elu), destinataire)
	}
}

func RecevoirMessageRouge(msg string) {
	k, _ := strconv.Atoi(findval(msg, MsgData))
	if elu == k {

		nbVoisinsAttendus--
		if nbVoisinsAttendus == 0 {
			if elu == MyId {
				win = true
				stderr.Println(orange, "["+Nom+"]", "J'ai gagné", raz)
			} else {
				// j'envoie rouge à mon parent
				destinataire := []int{parentTmp}
				envoyerA(election, rouge, strconv.Itoa(elu), destinataire)
			}
		}
	}
}

func recevoirMessageElection(msg string) {
	destinataires := findval(msg, MsgDestination)
	if destinataires != "" && !slices.Contains(strToIntTab(destinataires), MyId) {
		// Ce message ne m'était pas déstiné
		return
	}
	switch findval(msg, MsgType) {
	case bleu:
		RecevoirMessageBleu(msg)
		break
	case rouge:
		RecevoirMessageRouge(msg)
		break
	}
}

func resetElection() {
	elu = math.MaxInt
	nbVoisinsAttendus = NbVoisins
	parentTmp = 0
	win = false
}
