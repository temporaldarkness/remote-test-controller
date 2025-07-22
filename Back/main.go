package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Key     string `json:"key"`
}

type State struct {
	Running     bool      `json:"running"`
	StartTime   time.Time `json:"startTime,omitempty"`
	PausedAt    time.Time `json:"pausedAt,omitempty"` // Новое поле для времени паузы
	RPM         int       `json:"rpm"`
	Temperature float64   `json:"temperature"`
	Test        string    `json:"test"` // Номер / название испытания
	Name        string    `json:"name"` // Название установки
}

var (
	upgrader   = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	state      = State{}
	stateMutex sync.Mutex

	infoLog     = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLog    = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	hardwareLog = log.New(os.Stdout, "HARDWARE: ", log.Ldate|log.Ltime|log.Lshortfile)
	
	// Переменные конфига
	serverName    = "Sample Test Object"
	serverAddress = ":8080"
	serverKey     = ""
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		errorLog.Printf("Connection upgrade error: %v", err)
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
		
		// Проверка сообщения на ключ
		key, ok := req["key"].(string)
		if !ok {
			errorLog.Printf("[%s] Invalid key type: %T", conn.RemoteAddr(), req["key"])
			continue
		}
		if key != serverKey {
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
		case "status":
			infoLog.Printf("[%s] Performing action: Status", conn.RemoteAddr())
		case "ping":
			break
		case "changeTest":
			infoLog.Printf("[%s] Performing action: Change Test Name", conn.RemoteAddr())
			
			test, ok := req["test"].(string)
			if ok && test != "" {
				state.Test = test
				
				infoLog.Printf("[%s] Changed test name to: [%s]", conn.RemoteAddr(), state.Test)
			} else {
				errorLog.Printf("[%s] Invalid test name type: %T", conn.RemoteAddr(), req["test"])
			}

		default:
			infoLog.Printf("[%s] Unknown action: %v", conn.RemoteAddr(), action)
		}
		
		if state.Running && state.PausedAt.IsZero() { // Работает
			state.RPM = 1500 + (int(time.Now().Unix()) % 100) // Генерируем случайные обороты
			state.Temperature = 115.2 + (float64(time.Now().Unix()%10) / 10.0)
		} else { // Остановлен или на паузе
			state.RPM = 0
			state.Temperature = 25.0
		}
		resp := State{
			Running:     state.Running,
			StartTime:   state.StartTime,
			PausedAt:    state.PausedAt,
			RPM:         state.RPM,
			Temperature: state.Temperature,
			Name:        state.Name,
			Test:        state.Test,
		}
		stateMutex.Unlock()

		out := map[string]interface{}{
			"running":     resp.Running,
			"paused":      !resp.PausedAt.IsZero(),
			"rpm":         resp.RPM,
			"temperature": resp.Temperature,
			"name":        resp.Name,
			"test":        resp.Test,
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

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		errorLog.Printf("Config loading error: %v", err)
		return nil, err
	}
	defer file.Close()
	
	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		errorLog.Printf("Config decoding error: %v", err)
		return nil, err
	}
	return &cfg, err
}

func main() {
	
	// Загрузка конфига
	cfg, err := loadConfig("config.json")
	if err != nil {
		infoLog.Println("Unable to load config.json, falling back to defaults!")
	} else { // Проверка на пустые значения
		if cfg.Name != "" {
			serverName = cfg.Name
			state.Name = serverName // Установить название в состоянии
		} else {
			infoLog.Println("Value of field 'Name' not present in config, falling back to defaults!")
		}
		
		if cfg.Address != "" {
			serverAddress = cfg.Address
		} else {
			infoLog.Println("Value of field 'Address' not found in config, falling back to defaults!")
		}
		
		if cfg.Key != "" {
			serverKey = cfg.Key
		} else {
			infoLog.Println("Value of field 'Key' not found in config, falling back to defaults!")
		}
	}
	
	state.Test = "001" // Устанавливаем значение теста заранее
	
	infoLog.Println("Config data loaded!")
	
	if serverKey == "" {
		infoLog.Println("Note: An unset key is a large security risk!")
	}
	
	fs := http.FileServer(http.Dir(filepath.Join("..", "Front")))
	http.Handle("/", fs)
	http.HandleFunc("/ws", wsHandler)
	infoLog.Println("Running on ", serverAddress)
	errorLog.Fatal(http.ListenAndServe(serverAddress, nil))
}
