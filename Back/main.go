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
	hardwareLog   = log.New(os.Stdout, "HARDWARE: ", log.Ldate|log.Ltime|log.Lshortfile)
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		errorLog.Printf("[%s] Connection upgrade error: %v", conn.RemoteAddr(), err)
		return
	}
	defer conn.Close()
	
	conn.SetReadLimit(1024)
	
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			errorLog.Printf("[%s] Message reading error: %v", conn.RemoteAddr(), err)
			break
		}

		var req map[string]interface{}
		if err := json.Unmarshal(msg, &req); err != nil {
			errorLog.Printf("[%s] Message unmarshalling error: %v", conn.RemoteAddr(), err)
			continue
		}

		stateMutex.Lock()
		action, ok := req["action"].(string)
		if !ok {
			errorLog.Printf("[%s] Invalid action type: %T", conn.RemoteAddr(), req["action"])
			continue
		}
		
		switch action {
			case "start":
				if !state.Running {
					infoLog.Printf("[%s] Performing action: Start (Startup)", conn.RemoteAddr())
					state.Running = true
					state.StartTime = time.Now().UTC()
					state.PausedAt = time.Time{}
					hardwareStart()
				} else if !state.PausedAt.IsZero() {
					infoLog.Printf("[%s] Performing action: Start (Unpause)", conn.RemoteAddr())
					now := time.Now().UTC()
					state.StartTime = state.StartTime.Add(now.Sub(state.PausedAt))
					state.PausedAt = time.Time{}
					hardwareUnpause()
				}
			case "pause":
				if state.Running && state.PausedAt.IsZero() {
					infoLog.Printf("[%s] Performing action: Pause", conn.RemoteAddr())
					state.PausedAt = time.Now().UTC()
					hardwarePause()
				}
			case "stop":
				if state.Running {
					infoLog.Printf("[%s] Performing action: Stop", conn.RemoteAddr())
					state.Running = false
					state.StartTime = time.Time{}
					state.PausedAt = time.Time{}
					hardwareStop()
				}
			default:
				infoLog.Printf("[%s] Unknown action: %v", conn.RemoteAddr(), action)
		}
		resp := State{
			Running: state.Running,
			StartTime: state.StartTime,
			PausedAt: state.PausedAt,
		}
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
			errorLog.Printf("[%s] Response marshalling error: %v", conn.RemoteAddr(), err)
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			errorLog.Printf("[%s] Response writing error: %v", conn.RemoteAddr(), err)
		}
	}
}

func hardwareStart() {
	hardwareLog.Printf("Hardware test start!")
}

func hardwareStop() {
	hardwareLog.Printf("Hardware test stop!")
}

func hardwarePause() {
	hardwareLog.Printf("Hardware test pause!")
}

func hardwareUnpause() {
	hardwareLog.Printf("Hardware test unpause!")
}

func main() {
	fs := http.FileServer(http.Dir(filepath.Join("..", "Front")))
	http.Handle("/", fs)
	http.HandleFunc("/ws", wsHandler)
	infoLog.Println("Running on :8080!")
	errorLog.Fatal(http.ListenAndServe(":8080", nil))
}
