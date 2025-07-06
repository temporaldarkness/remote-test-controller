package main

import (
    "encoding/json"
    "log"
    "net/http"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

type State struct {
    Running   bool      `json:"running"`
    StartTime time.Time `json:"startTime,omitempty"`
}

var (
    upgrader = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }
    state      = State{}
    stateMutex sync.Mutex
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Upgrade error:", err)
        return
    }
    defer conn.Close()

    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            log.Println("Read error:", err)
            break
        }

        var req map[string]interface{}
        if err := json.Unmarshal(msg, &req); err != nil {
            continue
        }

        stateMutex.Lock()
        switch req["action"] {
        case "start":
            state.Running = true
            state.StartTime = time.Now()
        case "stop":
            state.Running = false
            state.StartTime = time.Time{}
        case "get_state":
            // ничего не меняем
        }
        resp := state
        stateMutex.Unlock()

        // Формируем ответ
        out := map[string]interface{}{
            "running": resp.Running,
        }
        if resp.Running {
            out["startTime"] = resp.StartTime.Format(time.RFC3339)
        }
        b, _ := json.Marshal(out)
        conn.WriteMessage(websocket.TextMessage, b)
    }
}

func main() {
    http.HandleFunc("/ws", wsHandler)
    log.Println("WebSocket server started at :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}