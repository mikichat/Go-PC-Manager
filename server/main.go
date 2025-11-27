package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 연결된 에이전트들을 저장하는 맵
	agents = make(map[*websocket.Conn]bool)
	// 연결된 대시보드들을 저장하는 맵
	dashboards = make(map[*websocket.Conn]bool)
	// 맵에 대한 동시 접근을 제어하기 위한 뮤텍스
	agentsMutex     = sync.Mutex{}
	dashboardsMutex = sync.Mutex{}
)

func main() {
	// 정적 파일 제공
	fs := http.FileServer(http.Dir("static"))
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
		log.Fatal(err)
	}
	defer ws.Close()

	agentsMutex.Lock()
	agents[ws] = true
	agentsMutex.Unlock()

	log.Println("New agent connected")

	// 에이전트 연결이 끊어졌을 때 맵에서 제거
	defer func() {
		agentsMutex.Lock()
		delete(agents, ws)
		agentsMutex.Unlock()
		log.Println("Agent disconnected")
	}()

	for {
		// 에이전트로부터 메시지 수신 (향후 구현)
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
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

	// 대시보드 연결이 끊어졌을 때 맵에서 제거
	defer func() {
		dashboardsMutex.Lock()
		delete(dashboards, ws)
		dashboardsMutex.Unlock()
		log.Println("Dashboard disconnected")
	}()

	for {
		// 대시보드로부터 메시지를 수신하여 모든 에이전트에게 전송
		_, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}

		agentsMutex.Lock()
		for agent := range agents {
			if err := agent.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println("write:", err)
			}
		}
		agentsMutex.Unlock()
	}
}
