package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 메시지 타입 정의
type MessageType string

const (
	MsgTypeRegister     MessageType = "register"
	MsgTypeCommand      MessageType = "command"
	MsgTypeCommandResult MessageType = "command_result"
	MsgTypeStatus       MessageType = "status"
	MsgTypeAgentList    MessageType = "agent_list"
	MsgTypeAgentUpdate  MessageType = "agent_update"
)

// 메시지 구조체
type Message struct {
	Type      MessageType              `json:"type"`
	AgentID   string                  `json:"agent_id,omitempty"`
	Command   string                  `json:"command,omitempty"`
	Result    *CommandResult          `json:"result,omitempty"`
	Status    *AgentStatus            `json:"status,omitempty"`
	AgentInfo *AgentInfo              `json:"agent_info,omitempty"`
	Agents    []AgentData             `json:"agents,omitempty"`
	Agent     *AgentData              `json:"agent,omitempty"`
}

// 명령 실행 결과
type CommandResult struct {
	Command   string `json:"command"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	ExitCode  int    `json:"exit_code"`
	Timestamp string `json:"timestamp"`
}

// 에이전트 상태 정보
type AgentStatus struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Uptime      int64   `json:"uptime"`
	Timestamp   string  `json:"timestamp"`
}

// 에이전트 정보
type AgentInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	MACAddr  string `json:"mac_addr"`
}

// 에이전트 데이터 (서버에서 관리)
type AgentData struct {
	ID        string       `json:"id"`
	Info      *AgentInfo   `json:"info"`
	Status    *AgentStatus `json:"status"`
	Conn      *websocket.Conn `json:"-"`
	LastSeen  time.Time    `json:"last_seen"`
	Connected bool         `json:"connected"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 개발 환경용, 프로덕션에서는 적절한 체크 필요
		},
	}

	// 연결된 에이전트들을 ID로 관리하는 맵
	agents = make(map[string]*AgentData)
	// 연결된 대시보드들을 저장하는 맵
	dashboards = make(map[*websocket.Conn]bool)
	// 맵에 대한 동시 접근을 제어하기 위한 뮤텍스
	agentsMutex     = sync.RWMutex{}
	dashboardsMutex = sync.Mutex{}
)

func main() {
	// 정적 파일 제공
	fs := http.FileServer(http.Dir("server/static"))
	http.Handle("/", fs)

	// 웹소켓 핸들러
	http.HandleFunc("/ws-agent", handleAgentConnections)
	http.HandleFunc("/ws-dashboard", handleDashboardConnections)

	// 서버 시작
	log.Println("http server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleAgentConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 업그레이드 실패: %v", err)
		return
	}
	defer ws.Close()

	var agentID string
	var agentData *AgentData

	// 에이전트 연결이 끊어졌을 때 맵에서 제거
	defer func() {
		if agentID != "" {
			agentsMutex.Lock()
			if agent, exists := agents[agentID]; exists && agent.Conn == ws {
				agent.Connected = false
				delete(agents, agentID)
				broadcastAgentUpdate(agentID, false)
				log.Printf("에이전트 연결 해제: %s", agentID)
			}
			agentsMutex.Unlock()
		}
	}()

	log.Println("새 에이전트 연결 시도")

	for {
		_, messageBytes, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("예상치 못한 연결 종료: %v", err)
			} else {
				log.Printf("메시지 수신 오류: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("메시지 파싱 실패: %v (원본: %s)", err, string(messageBytes))
			continue
		}

		switch msg.Type {
		case MsgTypeRegister:
			// 에이전트 등록
			if msg.AgentID == "" || msg.AgentInfo == nil {
				log.Printf("잘못된 등록 메시지: AgentID=%s, AgentInfo=%v", msg.AgentID, msg.AgentInfo)
				continue
			}
			agentID = msg.AgentID
			agentData = &AgentData{
				ID:        agentID,
				Info:      msg.AgentInfo,
				Conn:      ws,
				LastSeen:  time.Now(),
				Connected: true,
			}

			agentsMutex.Lock()
			agents[agentID] = agentData
			agentsMutex.Unlock()

			log.Printf("에이전트 등록됨: %s (%s, %s)", agentID, msg.AgentInfo.Hostname, msg.AgentInfo.OS)
			broadcastAgentUpdate(agentID, true)

		case MsgTypeStatus:
			// 상태 정보 업데이트
			if agentID == "" {
				log.Println("상태 업데이트: 에이전트 ID가 없음")
				continue
			}
			agentsMutex.Lock()
			if agent, exists := agents[agentID]; exists {
				agent.Status = msg.Status
				agent.LastSeen = time.Now()
			} else {
				log.Printf("상태 업데이트: 알 수 없는 에이전트 ID: %s", agentID)
			}
			agentsMutex.Unlock()
			broadcastAgentUpdate(agentID, true)

		case MsgTypeCommandResult:
			// 명령 실행 결과를 대시보드로 전달
			if msg.AgentID == "" {
				log.Println("명령 결과: 에이전트 ID가 없음")
				continue
			}
			log.Printf("명령 결과 수신: 에이전트=%s, 명령=%s", msg.AgentID, msg.Result.Command)
			broadcastToDashboards(msg)

		default:
			log.Printf("알 수 없는 메시지 타입: %s (에이전트: %s)", msg.Type, agentID)
		}
	}
}

func handleDashboardConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 업그레이드 실패: %v", err)
		return
	}
	defer ws.Close()

	dashboardsMutex.Lock()
	dashboards[ws] = true
	dashboardsMutex.Unlock()

	log.Println("새 대시보드 연결됨")

	// 연결 시 현재 에이전트 목록 전송
	if err := sendAgentList(ws); err != nil {
		log.Printf("에이전트 목록 전송 실패: %v", err)
	}

	// 대시보드 연결이 끊어졌을 때 맵에서 제거
	defer func() {
		dashboardsMutex.Lock()
		delete(dashboards, ws)
		dashboardsMutex.Unlock()
		log.Println("대시보드 연결 해제")
	}()

	for {
		_, messageBytes, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("예상치 못한 연결 종료: %v", err)
			} else {
				log.Printf("메시지 수신 오류: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("메시지 파싱 실패: %v (원본: %s)", err, string(messageBytes))
			continue
		}

		// 명령 메시지 처리
		if msg.Type == MsgTypeCommand && msg.Command != "" {
			if msg.AgentID != "" {
				// 특정 에이전트에 명령 전송
				log.Printf("명령 전송: 에이전트=%s, 명령=%s", msg.AgentID, msg.Command)
				sendCommandToAgent(msg.AgentID, msg.Command)
			} else {
				// 모든 에이전트에 명령 전송
				log.Printf("명령 브로드캐스트: 명령=%s", msg.Command)
				broadcastCommandToAgents(msg.Command)
			}
		} else {
			log.Printf("알 수 없는 메시지 타입: %s", msg.Type)
		}
	}
}

// 에이전트 목록을 대시보드에 전송
func sendAgentList(ws *websocket.Conn) error {
	agentsMutex.RLock()
	agentList := make([]AgentData, 0, len(agents))
	for _, agent := range agents {
		agentList = append(agentList, AgentData{
			ID:        agent.ID,
			Info:      agent.Info,
			Status:    agent.Status,
			LastSeen:  agent.LastSeen,
			Connected: agent.Connected,
		})
	}
	agentsMutex.RUnlock()

	msg := Message{
		Type:   MsgTypeAgentList,
		Agents: agentList,
	}
	if err := ws.WriteJSON(msg); err != nil {
		return fmt.Errorf("에이전트 목록 전송 실패: %w", err)
	}
	log.Printf("에이전트 목록 전송 완료: %d개", len(agentList))
	return nil
}

// 에이전트 업데이트를 모든 대시보드에 브로드캐스트
func broadcastAgentUpdate(agentID string, connected bool) {
	agentsMutex.RLock()
	agent, exists := agents[agentID]
	if !exists {
		agentsMutex.RUnlock()
		return
	}
	agentData := AgentData{
		ID:        agent.ID,
		Info:      agent.Info,
		Status:    agent.Status,
		LastSeen:  agent.LastSeen,
		Connected: connected,
	}
	agentsMutex.RUnlock()

	msg := Message{
		Type:  MsgTypeAgentUpdate,
		Agent: &agentData,
	}

	dashboardsMutex.Lock()
	for dashboard := range dashboards {
		if err := dashboard.WriteJSON(msg); err != nil {
			log.Println("Failed to broadcast agent update:", err)
		}
	}
	dashboardsMutex.Unlock()
}

// 대시보드에 메시지 브로드캐스트
func broadcastToDashboards(msg Message) {
	dashboardsMutex.Lock()
	for dashboard := range dashboards {
		if err := dashboard.WriteJSON(msg); err != nil {
			log.Println("Failed to broadcast to dashboard:", err)
		}
	}
	dashboardsMutex.Unlock()
}

// 특정 에이전트에 명령 전송
func sendCommandToAgent(agentID, command string) {
	agentsMutex.RLock()
	agent, exists := agents[agentID]
	agentsMutex.RUnlock()

	if !exists {
		log.Printf("명령 전송 실패: 에이전트를 찾을 수 없음 (ID: %s)", agentID)
		return
	}

	if !agent.Connected {
		log.Printf("명령 전송 실패: 에이전트가 연결되지 않음 (ID: %s)", agentID)
		return
	}

	msg := Message{
		Type:    MsgTypeCommand,
		Command: command,
	}

	if err := agent.Conn.WriteJSON(msg); err != nil {
		log.Printf("명령 전송 실패: 에이전트=%s, 오류=%v", agentID, err)
	} else {
		log.Printf("명령 전송 완료: 에이전트=%s, 명령=%s", agentID, command)
	}
}

// 모든 에이전트에 명령 브로드캐스트
func broadcastCommandToAgents(command string) {
	msg := Message{
		Type:    MsgTypeCommand,
		Command: command,
	}

	agentsMutex.RLock()
	count := 0
	for _, agent := range agents {
		if agent.Connected {
			if err := agent.Conn.WriteJSON(msg); err != nil {
				log.Printf("명령 브로드캐스트 실패: 에이전트=%s, 오류=%v", agent.ID, err)
			} else {
				count++
			}
		}
	}
	agentsMutex.RUnlock()
	log.Printf("명령 브로드캐스트 완료: %d개 에이전트에 전송", count)
}
