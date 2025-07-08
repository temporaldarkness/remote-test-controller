package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"
	"os"
)

type State struct {
	Running   bool      `json:"running"`
	StartTime time.Time `json:"startTime,omitempty"`
	PausedAt  time.Time `json:"pausedAt,omitempty"` // Новое поле для времени паузы
}

var (
	upgrader   = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	state      = State{}
	stateMutex sync.Mutex
	
	infoLog    = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLog   = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		errorLog.Println("Connection upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			errorLog.Println("Message reading error:", err)
			break
		}

		var req map[string]interface{}
		if err := json.Unmarshal(msg, &req); err != nil {
			errorLog.Println("Message unmarshalling error:", err)
			continue
		}

		stateMutex.Lock()
		infoLog.Println("Performing action:", req["action"])
		switch req["action"] {
		case "start":
			if !state.Running {
				state.Running = true
				state.StartTime = time.Now().UTC()
				state.PausedAt = time.Time{}
			} else if !state.PausedAt.IsZero() {
				state.StartTime = state.StartTime.Add(time.Since(state.PausedAt))
				state.PausedAt = time.Time{}
			}
		case "pause":
			if state.Running && state.PausedAt.IsZero() {
				state.PausedAt = time.Now().UTC()
			}
		case "stop":
			state.Running = false
			state.StartTime = time.Time{}
			state.PausedAt = time.Time{}
		}
		resp := state
		stateMutex.Unlock()

		out := map[string]interface{}{
			"running": resp.Running,
			"paused":  !resp.PausedAt.IsZero(),
		}
		if !resp.StartTime.IsZero() {
			out["startTime"] = resp.StartTime.Format(time.RFC3339)
		}
		b, err := json.Marshal(out)
		if err != nil {
			errorLog.Println("Response marshalling error:", err)
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			errorLog.Println("Response writing error:", err)
		}
	}
}

func main() {
	fs := http.FileServer(http.Dir(filepath.Join("..", "Front")))
	http.Handle("/", fs)
	http.HandleFunc("/ws", wsHandler)
	infoLog.Println("Running on :8080!")
	errorLog.Fatal(http.ListenAndServe(":8080", nil))
}
