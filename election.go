package main

import (
	"strconv"
	"strings"
)

var (
	elu               = int(^uint(0) >> 1)
	parent            = 0
	nbVoisinsAttendus = 0
)

func DemarrerElection() {
	if parent == 0 {
		elu = MyId
		parent = MyId
		msg := MsgFormat(MsgCategory, "bleu") + MsgFormat("k", strconv.Itoa(MyId))
		envoyerAuxVoisins(msg)
	}
}

func RecevoirMessageBleu(msg string) {
	kStr := findval(msg, "k")
	k, _ := strconv.Atoi(kStr)
	senderName := findval(msg, MsgDestination)
	senderId := getId(senderName)

	if k < elu {
		elu = k
		parent = senderId
		nbVoisinsAttendus = len(enfants)
		if nbVoisinsAttendus > 0 {
			envoyerAuxVoisins(msg)
		} else {
			msg = MsgFormat(MsgCategory, "rouge") + MsgFormat("k", strconv.Itoa(MyId))
			envoyer(msg)
		}
	} else if elu == k {
		msg = MsgFormat(MsgCategory, "rouge") + MsgFormat("k", strconv.Itoa(MyId))
		envoyer(msg)
	}
}
