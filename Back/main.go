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
	Fields  []Field `json:"fields"`
}

type Field struct {
	Id string   `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type StateField struct {
	Id string         `json:"id"`
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type State struct {
	Running   bool         `json:"running"`
	StartTime time.Time    `json:"startTime,omitempty"`
	PausedAt  time.Time    `json:"pausedAt,omitempty"` // Новое поле для времени паузы
	Fields    []StateField `json:"fields"`
	Test      string       `json:"test"` // Номер / название испытания
	Name      string       `json:"name"` // Название установки
}

var (
	upgrader   = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	states     = map[string]*State{} // Хранилище состояний по тестам
	stateMutex sync.Mutex

	infoLog     = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLog    = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	hardwareLog = log.New(os.Stdout, "HARDWARE: ", log.Ldate|log.Ltime|log.Lshortfile)
	
	// Переменные конфига
	serverName    = "Sample Test Object"
	serverAddress = ":8080"
	serverKey     = ""

	cfg           *Config // Глобальный конфиг
)

func getOrCreateState(test string, cfg *Config) *State {
	if s, ok := states[test]; ok {
		return s
	}
	// Создаём новое состояние для теста
	stateFields := make([]StateField, len(cfg.Fields))
	for i, field := range cfg.Fields {
		switch field.Type {
		case "string":
			stateFields[i] = StateField{field.Id, field.Name, field.Type, ""}
		case "int":
			stateFields[i] = StateField{field.Id, field.Name, field.Type, 0}
		case "float":
			stateFields[i] = StateField{field.Id, field.Name, field.Type, 0.0}
		default:
			stateFields[i] = StateField{field.Id, field.Name, field.Type, nil}
		}
	}
	s := &State{
		Running:   false,
		StartTime: time.Time{},
		PausedAt:  time.Time{},
		Fields:    stateFields,
		Test:      test,
		Name:      serverName,
	}
	states[test] = s
	return s
}

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
			stateMutex.Unlock()
			continue
		}

		// Определяем текущий тест
		test := "001"
		if t, ok := req["test"].(string); ok && t != "" {
			test = t
		}
		state := getOrCreateState(test, cfg)

		switch action {
		case "start":
			if !state.Running {
				infoLog.Printf("[%s] [Test %s] Performing action: Start (Startup)", conn.RemoteAddr(), state.Test)
				state.Running = true
				state.StartTime = time.Now().UTC()
				state.PausedAt = time.Time{}
				resetFieldsToDefault(state)
				hardwareStart()
			} else if !state.PausedAt.IsZero() {
				infoLog.Printf("[%s] [Test %s] Performing action: Start (Unpause)", conn.RemoteAddr(), state.Test)
				now := time.Now().UTC()
				state.StartTime = state.StartTime.Add(now.Sub(state.PausedAt))
				state.PausedAt = time.Time{}
				hardwareUnpause()
			}
		case "pause":
			if state.Running && state.PausedAt.IsZero() {
				infoLog.Printf("[%s] [Test %s] Performing action: Pause", conn.RemoteAddr(), state.Test)
				state.PausedAt = time.Now().UTC()
				hardwarePause()
			}
		case "stop":
			if state.Running {
				infoLog.Printf("[%s] [Test %s] Performing action: Stop", conn.RemoteAddr(), state.Test)
				state.Running = false
				state.StartTime = time.Time{}
				state.PausedAt = time.Time{}
				resetFieldsToDefault(state)
				hardwareStop()
			}
			// Всегда сбрасываем состояние после stop
			state.Running = false
			state.StartTime = time.Time{}
			state.PausedAt = time.Time{}
			resetFieldsToDefault(state)
		case "status":
			infoLog.Printf("[%s] [Test %s] Performing action: Status", conn.RemoteAddr(), state.Test)
		case "ping":
			break
		case "command":
			infoLog.Printf("[%s] [Test %s] Performing action: Command", conn.RemoteAddr(), state.Test)
			
			command, ok := req["command"].(string)
			if ok && command != "" {
				hardwareCommand(command)
			} else {
				errorLog.Printf("[%s] [Test %s] Invalid command type: %T", conn.RemoteAddr(), state.Test, req["command"])
			}

		default:
			infoLog.Printf("[%s] [Test %s] Unknown action: %v", conn.RemoteAddr(), state.Test, action)
		}
		
		hardwareUpdateFields(state) // Где-то здесь должно быть обращение к установке для получения её полей
		
		resp := *state
		stateMutex.Unlock()

		out := map[string]interface{}{
			"running": resp.Running,
			"paused":  !resp.PausedAt.IsZero(),
			"name":    resp.Name,
			"test":    resp.Test,
			"fields":  resp.Fields,
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

func hardwareCommand(command string) {
	hardwareLog.Printf("Hardware command accepted: [%s]!", command)
}

// В действительном исполнении эта функция должна быть подогнана под набор используемых полей, так что хардкод не большая проблема
func hardwareUpdateFields(s *State) {
	if s.Running && s.PausedAt.IsZero() {
		s.Fields[0].Value = 115.2 + (float64(time.Now().Unix()%10) / 10.0) // Temperature
		s.Fields[1].Value = 1500 + (int(time.Now().Unix()) % 100) // RPM
		s.Fields[2].Value = 100 + (int(time.Now().Unix() * 7) % 15) // Power
	} else {
		s.Fields[0].Value = 25.0 // Temperature
		s.Fields[1].Value = 0 // RPM
		s.Fields[2].Value = 0 // Power
	} 
	
	//hardwareLog.Printf("Hardware fields updated!")
}

func resetFieldsToDefault(s *State) {
	for i := range s.Fields {
		switch s.Fields[i].Type {
		case "string":
			s.Fields[i].Value = ""
		case "int":
			s.Fields[i].Value = 0
		case "float":
			s.Fields[i].Value = 0.0
		default:
			s.Fields[i].Value = nil
		}
	}
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
	var err error
	cfg, err = loadConfig("config.json")
	if err != nil || cfg == nil {
		infoLog.Println("Unable to load config.json, falling back to defaults!")
		cfg = &Config{
			Name:    serverName,
			Address: serverAddress,
			Key:     serverKey,
			Fields: []Field{
				{Id: "temperature", Name: "Температура, °C", Type: "float"},
				{Id: "rpm", Name: "Обороты, об/мин", Type: "int"},
				{Id: "power", Name: "Мощность, кВт", Type: "int"},
			},
		}
	} else {
		if cfg.Name != "" {
			serverName = cfg.Name
		}
		if cfg.Address != "" {
			serverAddress = cfg.Address
		}
		if cfg.Key != "" {
			serverKey = cfg.Key
		}
	}
	infoLog.Println("Config data loaded!")
	if serverKey == "" {
		infoLog.Println("Note: An unset key is a large security risk!")
	}
	// Инициализация первого теста
	states["001"] = getOrCreateState("001", cfg)
	// state = states["001"] // Удалено
	fs := http.FileServer(http.Dir(filepath.Join("..", "Front")))
	http.Handle("/", fs)
	http.HandleFunc("/ws", wsHandler)
	infoLog.Println("Running on ", serverAddress)
	errorLog.Fatal(http.ListenAndServe(serverAddress, nil))
}
