package main

import (
	"flag"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kardianos/service"
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

// Service setup
type program struct{}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	return nil
}

func main() {
	svcFlag := flag.String("service", "", "Control the system service.")
	flag.Parse()

	svcConfig := &service.Config{
		Name:        "GoPCAgent",
		DisplayName: "Go PC Agent",
		Description: "Agent for Go PC Management System",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if len(*svcFlag) != 0 {
		err := service.Control(s, *svcFlag)
		if err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
		return
	}

	err = s.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func (p *program) run() {
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
			// 연결이 끊어지면 재연결 시도 (서비스에서는 종료되지 않고 재시도해야 함)
			// 여기서는 간단히 함수 종료 후 서비스 재시작에 의존하거나
			// 내부 루프에서 재연결 로직을 구현해야 함.
			// 서비스 관리자가 재시작해주므로 일단 return
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

	status := AgentStatus{
		MemoryUsage: float64(m.Alloc) / 1024 / 1024, // MB
		CPUUsage:    0.0,
		DiskUsage:   0.0,
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
		"exit_code": 0,
	}

	msg := Message{
		Type:   "command_result",
		Result: result,
	}

	conn.WriteJSON(msg)
}
