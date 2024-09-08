package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
		fmt.Println("error upgrading websocket connection,", err)
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
		fmt.Println("error creating file:", err)
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
			fmt.Println("error writing to file:", err)
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
			fmt.Println("error with ListenAndServe: ", err)
		}
		fmt.Println("WebSocket server started on :8162")
	}()
}
