package main

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Transcript struct {
	identifier string `json:"-"`
	Username   string

	// TODO: timestamps
	Text      string
	Temporary string
}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	clients      = make(map[*websocket.Conn]bool)
	clientsMutex = sync.Mutex{}

	transcripts = make(map[string]Transcript)
)

func handleConnection(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Err(err).Msg("error upgrading websocket connection")
		return
	}
	defer ws.Close()

	clientsMutex.Lock()
	clients[ws] = true
	clientsMutex.Unlock()

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			clientsMutex.Lock()
			delete(clients, ws)
			clientsMutex.Unlock()
			break
		}
	}
}

func handleUpdates(transcriptChannel <-chan Transcript) {
	timestamp := time.Now().Format("20060102T150405Z")
	filename := "transcripts/transcript_" + timestamp + ".txt"
	file, err := os.Create(filename)
	if err != nil {
		log.Err(err).Msg("error creating file")
		return
	}
	defer file.Close()

	for {
		msg := <-transcriptChannel

		transcripts[msg.identifier] = Transcript{
			identifier: msg.identifier,

			Text:      transcripts[msg.identifier].Text + msg.Text,
			Temporary: msg.Temporary,
		}

		// Write transcript to file
		_, err := file.WriteString(msg.Text)
		if err != nil {
			log.Err(err).Msg("error writing to file")
		}

		clientsMutex.Lock()
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
		clientsMutex.Unlock()
	}
}

func manageTranscripts(transcriptChannel <-chan Transcript) {
	go handleUpdates(transcriptChannel)

	go func() {
		http.HandleFunc("/ws", handleConnection)

		fs := http.FileServer(http.Dir("./public"))
		http.Handle("/", fs)

		err := http.ListenAndServe(":8162", nil)
		if err != nil {
			log.Err(err).Msg("error with ListenAndServe")
		}
		log.Trace().Msg("WebSocket server started on :8162")
	}()
}
