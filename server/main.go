package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"gopc-server/config"
)

// 데이터 구조 정의
type AgentInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	MacAddr  string `json:"mac_addr"`
}

type AgentStatus struct {
	MemoryUsage float64 `json:"memory_usage"` // percent
	CPUUsage    float64 `json:"cpu_usage"`    // percent
	DiskUsage   float64 `json:"disk_usage"`   // percent
	Uptime      uint64  `json:"uptime"`       // seconds
}

type Agent struct {
	ID        string          `json:"id"` // WebSocket RemoteAddr as ID for now
	Conn      *websocket.Conn `json:"-"`
	Info      *AgentInfo      `json:"info"`
	Status    *AgentStatus    `json:"status"`
	LastSeen  time.Time       `json:"last_seen"`
	Connected bool            `json:"connected"`
}

// 메시지 타입 정의
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// 대시보드로 보낼 메시지
type DashboardMessage struct {
	Type   string      `json:"type"`
	Agents []*Agent    `json:"agents,omitempty"`
	Agent  *Agent      `json:"agent,omitempty"`
	Result interface{} `json:"result,omitempty"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 연결된 에이전트들을 저장하는 맵
	agents = make(map[*websocket.Conn]*Agent)
	// 연결된 대시보드들을 저장하는 맵
	dashboards = make(map[*websocket.Conn]bool)
	// 맵에 대한 동시 접근을 제어하기 위한 뮤텍스
	agentsMutex     = sync.Mutex{}
	dashboardsMutex = sync.Mutex{}
)

func main() {
	// 설정 로드
	cfg := config.Load()

	// 정적 파일 서빙
	fs := http.FileServer(http.Dir(cfg.StaticDir))
	http.Handle("/", fs)

	// 업데이트 파일 서빙
	http.Handle("/updates/", http.StripPrefix("/updates/", http.FileServer(http.Dir(cfg.UpdatesDir))))

	// 버전 확인 엔드포인트
	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		handleVersion(w, r, cfg.AgentVersion)
	})

	// 웹소켓 핸들러
	http.HandleFunc("/ws-agent", handleAgentConnections)
	http.HandleFunc("/ws-dashboard", handleDashboardConnections)

	// 서버 시작
	log.Printf("http server started on %s", cfg.GetListenAddr())
	err := http.ListenAndServe(cfg.GetListenAddr(), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleVersion(w http.ResponseWriter, r *http.Request, version string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"version": version,
	})
}

func handleAgentConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	agentID := ws.RemoteAddr().String()
	agent := &Agent{
		ID:        agentID,
		Conn:      ws,
		LastSeen:  time.Now(),
		Connected: true,
	}

	agentsMutex.Lock()
	agents[ws] = agent
	agentsMutex.Unlock()

	log.Printf("New agent connected: %s", agentID)

	// 에이전트 연결이 끊어졌을 때 처리
	defer func() {
		agentsMutex.Lock()
		if agent, ok := agents[ws]; ok {
			agent.Connected = false
			agent.LastSeen = time.Now()
			// 맵에서 완전히 삭제하지 않고 연결 끊김 상태로 유지하거나,
			// 재연결 시 식별을 위해 유지할 수 있음.
			// 여기서는 간단히 맵에서 제거하고 대시보드에 알림
			delete(agents, ws)

			// 대시보드에 연결 끊김 알림 (ID만 보내거나 상태 업데이트)
			// 실제 구현에서는 ID로 추적해야 하지만, 여기서는 객체를 보냄
			agent.Connected = false
			broadcastAgentUpdate(agent)
		}
		agentsMutex.Unlock()
		log.Println("Agent disconnected")
	}()

	for {
		var msg map[string]interface{}
		err := ws.ReadJSON(&msg)
		if err != nil {
			break
		}

		msgType, _ := msg["type"].(string)

		agentsMutex.Lock()
		agent.LastSeen = time.Now()

		switch msgType {
		case "register":
			infoData, _ := json.Marshal(msg["info"])
			var info AgentInfo
			json.Unmarshal(infoData, &info)
			agent.Info = &info
			broadcastAgentUpdate(agent)

		case "status":
			statusData, _ := json.Marshal(msg["status"])
			var status AgentStatus
			json.Unmarshal(statusData, &status)
			agent.Status = &status
			broadcastAgentUpdate(agent)

		case "command_result":
			// 대시보드에 결과 전달
			broadcastCommandResult(msg, agent.ID)
		}
		agentsMutex.Unlock()
	}
}

func handleDashboardConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	dashboardsMutex.Lock()
	dashboards[ws] = true
	dashboardsMutex.Unlock()

	log.Println("New dashboard connected")

	// 연결 즉시 현재 에이전트 목록 전송
	sendAgentList(ws)

	defer func() {
		dashboardsMutex.Lock()
		delete(dashboards, ws)
		dashboardsMutex.Unlock()
		log.Println("Dashboard disconnected")
	}()

	for {
		// 대시보드로부터 메시지를 수신 (명령 등)
		var msg map[string]interface{}
		err := ws.ReadJSON(&msg)
		if err != nil {
			break
		}

		if msg["type"] == "command" {
			handleCommand(msg)
		}
	}
}

func sendAgentList(ws *websocket.Conn) {
	agentsMutex.Lock()
	defer agentsMutex.Unlock()

	list := make([]*Agent, 0, len(agents))
	for _, agent := range agents {
		list = append(list, agent)
	}

	msg := DashboardMessage{
		Type:   "agent_list",
		Agents: list,
	}
	ws.WriteJSON(msg)
}

func broadcastAgentUpdate(agent *Agent) {
	msg := DashboardMessage{
		Type:  "agent_update",
		Agent: agent,
	}
	broadcastToDashboards(msg)
}

func broadcastCommandResult(originalMsg map[string]interface{}, agentID string) {
	// 원본 메시지에 agent_id 추가하여 전송
	resultMsg := map[string]interface{}{
		"type":     "command_result",
		"agent_id": agentID,
		"result":   originalMsg["result"],
	}
	broadcastToDashboards(resultMsg)
}

func broadcastToDashboards(msg interface{}) {
	dashboardsMutex.Lock()
	defer dashboardsMutex.Unlock()

	for ws := range dashboards {
		err := ws.WriteJSON(msg)
		if err != nil {
			log.Println("write to dashboard error:", err)
			ws.Close()
			delete(dashboards, ws)
		}
	}
}

func handleCommand(msg map[string]interface{}) {
	command, _ := msg["command"].(string)
	targetAgentID, _ := msg["agent_id"].(string)

	// 설정 로드 (토큰 가져오기 위해)
	cfg := config.Load()

	agentsMutex.Lock()
	defer agentsMutex.Unlock()

	// 명령 메시지 생성 (JSON)
	cmdMsg := map[string]string{
		"token":   cfg.AuthToken,
		"command": command,
	}
	cmdBytes, _ := json.Marshal(cmdMsg)

	for _, agent := range agents {
		// 특정 에이전트에게만 전송하거나 전체 전송
		if targetAgentID == "" || agent.ID == targetAgentID {
			// JSON 형태로 전송
			err := agent.Conn.WriteMessage(websocket.TextMessage, cmdBytes)
			if err != nil {
				log.Println("write to agent error:", err)
			}
		}
	}
}
