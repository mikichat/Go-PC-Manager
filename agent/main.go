package main

import (
	"log"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	serverAddr := "localhost:8080"
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: "/ws-agent"}
	log.Printf("connecting to %s", u.String())

	var conn *websocket.Conn
	var err error

	// 연결될 때까지 5초 간격으로 재시도
	for {
		log.Println("Attempting to connect...")
		conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Println("dial error:", err)
			log.Println("Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
	defer conn.Close()
	log.Println("Connected to server")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		log.Printf("recv: %s", message)

		// 운영 체제에 따라 다른 명령 실행
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", string(message))
		} else {
			cmd = exec.Command("sh", "-c", string(message))
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Println("exec error:", err)
		}
		log.Println("output:", string(output))
	}
}
