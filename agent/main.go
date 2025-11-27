package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
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
)

// 메시지 구조체
type Message struct {
	Type      MessageType              `json:"type"`
	AgentID   string                  `json:"agent_id,omitempty"`
	Command   string                  `json:"command,omitempty"`
	Result    *CommandResult          `json:"result,omitempty"`
	Status    *AgentStatus            `json:"status,omitempty"`
	AgentInfo *AgentInfo              `json:"agent_info,omitempty"`
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

// 고유 에이전트 ID 생성 (호스트명 + MAC 주소)
func getAgentID() (string, *AgentInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", nil, err
	}

	// MAC 주소 가져오기
	macAddr := getMACAddress()
	agentID := hostname + "-" + macAddr

	info := &AgentInfo{
		Hostname: hostname,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		MACAddr:  macAddr,
	}

	return agentID, info, nil
}

// MAC 주소 가져오기
func getMACAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}

	for _, iface := range interfaces {
		if iface.HardwareAddr != nil && len(iface.HardwareAddr) > 0 {
			return iface.HardwareAddr.String()
		}
	}
	return "unknown"
}

var startTime = time.Now() // 프로그램 시작 시간

func main() {
	serverAddr := "localhost:8080"
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: "/ws-agent"}
	log.Printf("서버에 연결 시도: %s", u.String())

	// 에이전트 ID 및 정보 생성
	agentID, agentInfo, err := getAgentID()
	if err != nil {
		log.Fatalf("에이전트 ID 생성 실패: %v", err)
	}
	log.Printf("에이전트 ID: %s", agentID)

	// 무한 재연결 루프
	for {
		if err := runAgent(u, agentID, agentInfo); err != nil {
			log.Printf("연결 오류 발생: %v", err)
			log.Println("5초 후 재연결 시도...")
			time.Sleep(5 * time.Second)
		}
	}
}

// 에이전트 실행 (연결 및 통신 처리)
func runAgent(u url.URL, agentID string, agentInfo *AgentInfo) error {
	// 연결 시도
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("서버 연결 실패: %w", err)
	}
	defer conn.Close()
	log.Println("서버에 연결됨")

	// 서버에 에이전트 등록
	registerMsg := Message{
		Type:      MsgTypeRegister,
		AgentID:   agentID,
		AgentInfo: agentInfo,
	}
	if err := conn.WriteJSON(registerMsg); err != nil {
		return fmt.Errorf("서버 등록 실패: %w", err)
	}
	log.Println("서버에 등록됨")

	// 상태 모니터링을 위한 틱커 (30초마다)
	statusTicker := time.NewTicker(30 * time.Second)
	defer statusTicker.Stop()

	// 종료 신호 채널
	done := make(chan error, 1)

	// 메시지 수신 루프
	go func() {
		for {
			_, messageBytes, err := conn.ReadMessage()
			if err != nil {
				done <- fmt.Errorf("메시지 수신 오류: %w", err)
				return
			}

			var msg Message
			if err := json.Unmarshal(messageBytes, &msg); err != nil {
				log.Printf("메시지 파싱 실패: %v (원본: %s)", err, string(messageBytes))
				continue
			}

			// 명령 메시지 처리
			if msg.Type == MsgTypeCommand && msg.Command != "" {
				log.Printf("명령 수신: %s", msg.Command)
				go executeCommand(conn, agentID, msg.Command)
			} else {
				log.Printf("알 수 없는 메시지 타입: %s", msg.Type)
			}
		}
	}()

	// 상태 정보 주기적 전송
	go func() {
		for {
			select {
			case <-statusTicker.C:
				status := collectStatus()
				statusMsg := Message{
					Type:    MsgTypeStatus,
					AgentID: agentID,
					Status:  status,
				}
				if err := conn.WriteJSON(statusMsg); err != nil {
					done <- fmt.Errorf("상태 전송 실패: %w", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// 오류 대기
	return <-done
}

// 명령 실행 및 결과 전송
func executeCommand(conn *websocket.Conn, agentID, command string) {
	log.Printf("명령 실행 시작: %s", command)
	
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	exitCode := 0
	errorMsg := ""

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
			errorMsg = fmt.Sprintf("명령 실행 실패 (종료 코드: %d)", exitCode)
		} else {
			exitCode = -1
			errorMsg = fmt.Sprintf("명령 실행 오류: %v", err)
		}
		log.Printf("명령 실행 오류: %s", errorMsg)
	} else {
		log.Printf("명령 실행 완료 (종료 코드: %d)", exitCode)
	}

	result := &CommandResult{
		Command:   command,
		Output:    string(output),
		Error:     errorMsg,
		ExitCode:  exitCode,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	resultMsg := Message{
		Type:    MsgTypeCommandResult,
		AgentID: agentID,
		Result:  result,
	}

	if err := conn.WriteJSON(resultMsg); err != nil {
		log.Printf("명령 결과 전송 실패: %v", err)
	} else {
		log.Printf("명령 결과 전송 완료")
	}
}

// 시스템 상태 수집
func collectStatus() *AgentStatus {
	cpuUsage := getCPUUsage()
	memoryUsage := getMemoryUsage()
	diskUsage := getDiskUsage()
	uptime := int64(time.Since(startTime).Seconds())

	return &AgentStatus{
		CPUUsage:    cpuUsage,
		MemoryUsage: memoryUsage,
		DiskUsage:   diskUsage,
		Uptime:      uptime,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// CPU 사용률 가져오기 (Windows)
func getCPUUsage() float64 {
	if runtime.GOOS == "windows" {
		// PowerShell을 사용하여 CPU 사용률 가져오기
		cmd := exec.Command("powershell", "-Command", 
			"(Get-Counter '\\Processor(_Total)\\% Processor Time').CounterSamples[0].CookedValue")
		output, err := cmd.Output()
		if err != nil {
			log.Printf("CPU 사용률 조회 실패: %v", err)
			return 0.0
		}
		usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
		if err != nil {
			log.Printf("CPU 사용률 파싱 실패: %v (출력: %s)", err, string(output))
			return 0.0
		}
		return usage
	}
	// Linux/Mac의 경우 (향후 구현)
	return 0.0
}

// 메모리 사용률 가져오기 (Windows)
func getMemoryUsage() float64 {
	if runtime.GOOS == "windows" {
		// PowerShell을 사용하여 메모리 사용률 가져오기
		cmd := exec.Command("powershell", "-Command", 
			"$mem = Get-CimInstance Win32_OperatingSystem; [math]::Round((($mem.TotalVisibleMemorySize - $mem.FreePhysicalMemory) / $mem.TotalVisibleMemorySize) * 100, 2)")
		output, err := cmd.Output()
		if err != nil {
			log.Printf("메모리 사용률 조회 실패: %v", err)
			return 0.0
		}
		usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
		if err != nil {
			log.Printf("메모리 사용률 파싱 실패: %v (출력: %s)", err, string(output))
			return 0.0
		}
		return usage
	}
	// Linux/Mac의 경우 (향후 구현)
	return 0.0
}

// 디스크 사용률 가져오기 (Windows - C: 드라이브)
func getDiskUsage() float64 {
	if runtime.GOOS == "windows" {
		// PowerShell을 사용하여 C: 드라이브 사용률 가져오기
		cmd := exec.Command("powershell", "-Command", 
			"$disk = Get-CimInstance Win32_LogicalDisk -Filter \"DeviceID='C:'\"; [math]::Round((($disk.Size - $disk.FreeSpace) / $disk.Size) * 100, 2)")
		output, err := cmd.Output()
		if err != nil {
			log.Printf("디스크 사용률 조회 실패: %v", err)
			return 0.0
		}
		usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
		if err != nil {
			log.Printf("디스크 사용률 파싱 실패: %v (출력: %s)", err, string(output))
			return 0.0
		}
		return usage
	}
	// Linux/Mac의 경우 (향후 구현)
	return 0.0
}
