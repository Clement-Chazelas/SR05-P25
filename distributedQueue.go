package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type MessageType string

// Les différents types de messages qu'on peut recevoir
const (
	Request        MessageType = "requête"
	Release        MessageType = "libération"
	Acknowledgment MessageType = "accusé"
)

type Message struct {
	Type MessageType `json:"type"`
	Date int         `json:"date"`
	From int         `json:"from"`
}

type Site struct {
	ID  int       //son id
	N   int       //nombre de sites
	Tab []Message //tableau des messsages
	Hi  int       //horloge
}

func NewSite(id, n int) *Site {
	site := &Site{
		ID:  id,
		N:   n,
		Tab: make([]Message, n),
		Hi:  0,
	}
	for i := 0; i < n; i++ {
		site.Tab[i] = Message{Type: Release, Date: 0}
	}
	return site
}

func (s *Site) Run() {
	go s.readFromStdin()
}

func (s *Site) handleMessage(msg Message) {
	s.Hi = max(s.Hi, msg.Date) + 1 //On augmente l'horloge logique, tout en s'assurant qu'elle est supérieure à celle du message reçu (qu'elle soit correcte)
	if msg.Type == Request {       // Si c'est une requête on applique le bon traitement
		s.Tab[msg.From] = msg
		s.sendMessage(Message{Type: Acknowledgment, Date: s.Hi, From: s.ID})

		// Vérification si la requête locale est la plus ancienne
		if s.Tab[s.ID].Type == Request && s.isOldestRequest() {
			s.sendMessage(Message{Type: "débutSC", Date: s.Hi, From: s.ID})
		}
	} else if msg.Type == Release { // Si c'est une libération on applique le bon traitement
		s.Tab[msg.From] = msg
	} else if msg.Type == Acknowledgment { // Si c'est un accusé de réception on applique le bon traitement
		if s.Tab[msg.From].Type != Request {
			s.Tab[msg.From] = msg
		}
		// Vérification si la requête locale est la plus ancienne
		if s.Tab[s.ID].Type == Request && s.isOldestRequest() {
			s.sendMessage(Message{Type: "débutSC", Date: s.Hi, From: s.ID})
		}
	}
}

func (s *Site) handleSC(msg string) { // Fonction pour gérer les messages de demande de section critique
	switch msg {
	case "demandeSC": //Si c'est une demande de section critique on applique le bon traitement
		s.Hi++
		req := Message{Type: Request, Date: s.Hi, From: s.ID}
		s.Tab[s.ID] = req
		s.sendMessage(req)
	case "finSC": // Si c'est une fin de section critique on applique le bon traitement
		s.Hi++
		rel := Message{Type: Release, Date: s.Hi, From: s.ID}
		s.Tab[s.ID] = rel
		s.sendMessage(rel)
	default:
		fmt.Fprintf(os.Stderr, "Demande inconnue : %s\n", msg)
	}
}

// isOldestRequest vérifie si la requête locale est la plus ancienne
func (s *Site) isOldestRequest() bool {
	for k := 0; k < s.N; k++ {
		if k != s.ID {
			if s.Tab[k].Type == Request && (s.Tab[k].Date < s.Tab[s.ID].Date || (s.Tab[k].Date == s.Tab[s.ID].Date && k < s.ID)) {
				return false
			}
		}
	}
	return true
}

// sendMessage envoie un message au site distant, à voir si on change pas le format de l'envoi
func (s *Site) sendMessage(msg Message) {
	data, _ := json.Marshal(msg)
	fmt.Println(string(data))
}

// readFromStdin lit les messages depuis l'entrée standard
func (s *Site) readFromStdin() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var msg Message
		text := scanner.Text()
		if len(text) >= 4 && text[:4] == "APP:" { //On vérifie si c'est quelque chose qui touche à la section critique
			s.handleSC(text[4:])
			continue
		}

		if err := json.Unmarshal([]byte(scanner.Text()), &msg); err == nil { //Dans le cas où c'est un message JSON
			s.handleMessage(msg)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
