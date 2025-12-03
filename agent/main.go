package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"image/png"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kardianos/service"
	"github.com/kbinani/screenshot"

	"gopc-agent/config"
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
	// 설정 로드
	cfg := config.Load()

	// 로그 파일 설정
	exePath, err := os.Executable()
	if err == nil {
		logDir := filepath.Dir(exePath)
		logFile := filepath.Join(logDir, cfg.LogFile)
		f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err == nil {
			defer f.Close()
			log.SetOutput(f)
		}
	}

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
	// 설정 로드
	cfg := config.Load()

	u := url.URL{Scheme: "ws", Host: cfg.ServerAddress, Path: "/ws-agent"}
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

	// 상태 업데이트 및 버전 확인 고루틴
	go func() {
		ticker := time.NewTicker(cfg.GetStatusDuration())
		updateTicker := time.NewTicker(cfg.GetUpdateCheckDuration())
		defer ticker.Stop()
		defer updateTicker.Stop()

		for {
			select {
			case <-ticker.C:
				sendStatus(conn)
			case <-updateTicker.C:
				checkForUpdates(cfg.ServerAddress)
			}
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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in executeCommand: %v", r)
		}
	}()

	// GUI 명령 확인 (gui: 접두사)
	if len(command) > 4 && command[:4] == "gui:" {
		guiCmd := command[4:]
		log.Printf("Executing GUI command: %s", guiCmd)
		err := runAsUser(guiCmd)

		resultMsg := ""
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
			log.Printf("GUI execution error: %v", err)
		} else {
			resultMsg = "GUI command launched successfully"
		}

		result := map[string]interface{}{
			"command":   command,
			"output":    resultMsg,
			"error":     errorMsg,
			"timestamp": time.Now(),
			"exit_code": 0,
		}

		msg := Message{
			Type:   "command_result",
			Result: result,
		}
		conn.WriteJSON(msg)
		return
	}

	// 스크린샷 명령 확인
	if command == "capture_screen" {
		log.Println("Executing screenshot capture...")

		// 화면 캡처
		n := screenshot.NumActiveDisplays()
		if n <= 0 {
			log.Println("No active displays found")
			return
		}

		// 주 모니터(0번) 캡처
		bounds := screenshot.GetDisplayBounds(0)
		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			log.Printf("Screenshot failed: %v", err)
			return
		}

		// 이미지를 PNG로 인코딩하여 버퍼에 저장
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		if err != nil {
			log.Printf("PNG encoding failed: %v", err)
			return
		}

		// Base64 인코딩
		encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

		// 결과 전송
		result := map[string]interface{}{
			"command":    command,
			"image_data": encoded,
			"timestamp":  time.Now(),
		}

		msg := Message{
			Type:   "command_result", // 기존 타입 재사용 또는 screenshot_result 사용
			Result: result,
		}

		// 메시지 타입이 command_result이면 서버가 자동으로 중계함
		// 하지만 대시보드에서 구분을 위해 result 내부에 image_data가 있으면 스크린샷으로 처리하도록 할 수 있음
		// 또는 별도 타입을 사용할 수도 있음. 여기서는 command_result 사용하고 payload에 image_data 포함

		conn.WriteJSON(msg)
		log.Println("Screenshot sent successfully")
		return
	}

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
