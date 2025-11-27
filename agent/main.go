package main

import (
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
)

type AgentInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	MacAddr  string `json:"mac_addr"`
}

type AgentStatus struct {
	MemoryUsage float64 `json:"memory_usage"`
	CPUUsage    float64 `json:"cpu_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Uptime      uint64  `json:"uptime"`
}

type Message struct {
	Type   string       `json:"type"`
	Info   *AgentInfo   `json:"info,omitempty"`
	Status *AgentStatus `json:"status,omitempty"`
	Result interface{}  `json:"result,omitempty"`
}

var startTime = time.Now()

func main() {
	serverAddr := "localhost:8080"
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: "/ws-agent"}
	log.Printf("connecting to %s", u.String())

	var conn *websocket.Conn
	var err error

	// 연결 재시도 루프
	for {
		conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Println("dial error:", err)
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
	defer conn.Close()
	log.Println("Connected to server")

	// 등록 메시지 전송
	sendRegister(conn)

	// 상태 업데이트 고루틴
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			sendStatus(conn)
		}
	}()

	// 메시지 수신 루프
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		log.Printf("recv: %s", message)

		// 명령 실행
		go executeCommand(conn, string(message))
	}
}

func sendRegister(conn *websocket.Conn) {
	hostname, _ := os.Hostname()

	// MAC 주소 가져오기
	macAddr := ""
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			if iface.HardwareAddr != nil {
				macAddr = iface.HardwareAddr.String()
				break
			}
		}
	}

	info := AgentInfo{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		MacAddr:  macAddr,
	}

	msg := Message{
		Type: "register",
		Info: &info,
	}

	conn.WriteJSON(msg)
}

func sendStatus(conn *websocket.Conn) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 간단한 시뮬레이션 값 (실제 구현은 OS별 라이브러리 필요)
	status := AgentStatus{
		MemoryUsage: float64(m.Alloc) / 1024 / 1024, // MB
		CPUUsage:    0.0,                            // 구현 복잡성으로 생략
		DiskUsage:   0.0,                            // 구현 복잡성으로 생략
		Uptime:      uint64(time.Since(startTime).Seconds()),
	}

	msg := Message{
		Type:   "status",
		Status: &status,
	}

	conn.WriteJSON(msg)
}

func executeCommand(conn *websocket.Conn, command string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	resultOutput := string(output)
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	result := map[string]interface{}{
		"command":   command,
		"output":    resultOutput,
		"error":     errorMsg,
		"timestamp": time.Now(),
		"exit_code": 0, // 간단화
	}

	msg := Message{
		Type:   "command_result",
		Result: result,
	}

	conn.WriteJSON(msg)
}
