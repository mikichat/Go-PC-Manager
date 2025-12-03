package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

//go:embed agent.exe
var agentExe []byte

const (
	InstallDir = "C:\\Program Files\\GoPCManager"
	AgentFile  = "agent.exe"
	ConfigFile = "config.yaml"
)

func main() {
	// 관리자 권한 확인 및 승격
	if !isAdmin() {
		fmt.Println("관리자 권한을 요청합니다...")
		if err := runMeElevated(); err != nil {
			fmt.Printf("관리자 권한 요청 실패: %v\n", err)
			pause()
			return
		}
		return
	}

	for {
		clearScreen()
		fmt.Println("=========================================")
		fmt.Println("    Go PC Agent 설치/삭제 관리자")
		fmt.Println("=========================================")
		fmt.Println("1. 에이전트 서비스 설치 및 시작")
		fmt.Println("2. 에이전트 서비스 중지 및 제거")
		fmt.Println("3. 종료")
		fmt.Println("=========================================")
		fmt.Print("선택하세요 (1-3): ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			installService()
		case "2":
			uninstallService()
		case "3":
			fmt.Println("프로그램을 종료합니다.")
			return
		default:
			fmt.Println("잘못된 입력입니다. 다시 선택해주세요.")
			pause()
		}
	}
}

func installService() {
	fmt.Println("\n[설치 진행 중...]")

	// 설치 디렉토리 생성
	fmt.Printf("- 설치 폴더 생성 중 (%s)...\n", InstallDir)
	if err := os.MkdirAll(InstallDir, 0755); err != nil {
		fmt.Printf("실패: 폴더 생성 오류 - %v\n", err)
		pause()
		return
	}

	// agent.exe 파일 추출
	targetPath := filepath.Join(InstallDir, AgentFile)
	fmt.Println("- 파일 추출 중...")
	if err := os.WriteFile(targetPath, agentExe, 0755); err != nil {
		fmt.Printf("실패: 파일 쓰기 오류 - %v\n", err)
		pause()
		return
	}

	// config.yaml 생성 (기본값)
	configPath := filepath.Join(InstallDir, ConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("- 기본 설정 파일 생성 중...")
		defaultConfig := `server_address: "localhost:8080"
status_interval: 5
update_check_interval: 60
log_file: "agent.log"
`
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			fmt.Printf("경고: 설정 파일 생성 실패 - %v\n", err)
		}
	}

	// 서비스 설치
	fmt.Println("- 서비스 등록 중...")
	if err := runCommand(targetPath, "-service", "install"); err != nil {
		fmt.Printf("실패: %v\n", err)
		pause()
		return
	}

	// 서비스 시작
	fmt.Println("- 서비스 시작 중...")
	if err := runCommand(targetPath, "-service", "start"); err != nil {
		fmt.Printf("실패: %v\n", err)
		pause()
		return
	}

	fmt.Println("\n✅ 에이전트가 성공적으로 설치되고 시작되었습니다!")
	fmt.Printf("설치 위치: %s\n", InstallDir)
	pause()
}

func uninstallService() {
	fmt.Println("\n[제거 진행 중...]")

	targetPath := filepath.Join(InstallDir, AgentFile)

	// 파일 존재 확인
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		fmt.Println("오류: 설치된 에이전트 파일을 찾을 수 없습니다.")
		fmt.Printf("경로: %s\n", targetPath)

		// 파일이 없어도 서비스가 남아있을 수 있으므로 sc 명령어로 시도
		fmt.Println("- sc 명령어로 서비스 제거 시도...")
		exec.Command("sc", "stop", "GoPCAgent").Run()
		exec.Command("sc", "delete", "GoPCAgent").Run()
	} else {
		// 서비스 중지
		fmt.Println("- 서비스 중지 중...")
		runCommand(targetPath, "-service", "stop")

		// 서비스 제거
		fmt.Println("- 서비스 제거 중...")
		if err := runCommand(targetPath, "-service", "uninstall"); err != nil {
			fmt.Printf("실패: 서비스 제거 오류 - %v\n", err)
			// 계속 진행 (파일 삭제 시도)
		}
	}

	// 잠시 대기 (프로세스 해제)
	time.Sleep(1 * time.Second)

	// 파일 및 폴더 삭제
	fmt.Println("- 설치 파일 및 폴더 삭제 중...")
	if err := os.RemoveAll(InstallDir); err != nil {
		fmt.Printf("경고: 폴더 삭제 실패 (수동 삭제 필요) - %v\n", err)
	} else {
		fmt.Println("폴더 삭제 완료.")
	}

	fmt.Println("\n✅ 에이전트가 성공적으로 제거되었습니다!")
	pause()
}

func runCommand(name string, args ...string) error {
	fmt.Printf("DEBUG: Executing: %s %v\n", name, args)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 실행 파일이 있는 디렉토리를 작업 디렉토리로 설정
	dir := filepath.Dir(name)
	if filepath.IsAbs(dir) {
		cmd.Dir = dir
	}

	return cmd.Run()
}

func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

func runMeElevated() error {
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verb := "runas"
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argsPtr, _ := syscall.UTF16PtrFromString(args)
	verbPtr, _ := syscall.UTF16PtrFromString(verb)

	showCmd := int32(windows.SW_NORMAL)

	err := windows.ShellExecute(0, verbPtr, exePtr, argsPtr, cwdPtr, showCmd)
	if err != nil {
		return err
	}
	return nil
}

func clearScreen() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func pause() {
	fmt.Println("\n계속하려면 엔터 키를 누르세요...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
