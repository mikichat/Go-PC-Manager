package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
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

	// agent.exe 존재 확인
	if _, err := os.Stat("agent.exe"); os.IsNotExist(err) {
		fmt.Println("오류: agent.exe 파일을 찾을 수 없습니다.")
		fmt.Println("setup.exe는 agent.exe와 같은 폴더에 있어야 합니다.")
		pause()
		return
	}

	// 서비스 설치
	fmt.Println("- 서비스 등록 중...")
	if err := runCommand("agent.exe", "-service", "install"); err != nil {
		fmt.Printf("실패: %v\n", err)
		pause()
		return
	}

	// 서비스 시작
	fmt.Println("- 서비스 시작 중...")
	if err := runCommand("agent.exe", "-service", "start"); err != nil {
		fmt.Printf("실패: %v\n", err)
		pause()
		return
	}

	fmt.Println("\n✅ 에이전트가 성공적으로 설치되고 시작되었습니다!")
	pause()
}

func uninstallService() {
	fmt.Println("\n[제거 진행 중...]")

	// 서비스 중지
	fmt.Println("- 서비스 중지 중...")
	runCommand("agent.exe", "-service", "stop") // 실패해도 계속 진행

	// 서비스 제거
	fmt.Println("- 서비스 제거 중...")
	if err := runCommand("agent.exe", "-service", "uninstall"); err != nil {
		fmt.Printf("실패: %v\n", err)
		pause()
		return
	}

	fmt.Println("\n✅ 에이전트가 성공적으로 제거되었습니다!")
	pause()
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
