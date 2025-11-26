# 🖥️ Go PC Management System (Academy Edition)

학원 및 교육장의 다수 PC를 중앙에서 효율적으로 관리하기 위한 **Go(Golang) 기반의 원격 관리 시스템**입니다.  
가볍고 빠른 **Go 언어**의 장점을 활용하여, 단일 실행 파일로 배포되며 시스템 리소스를 최소화하도록 설계되었습니다.

---

## 📋 프로젝트 개요 (Overview)

* **목표:** 중앙 서버에서 다수의 클라이언트(학생용 PC) 상태를 모니터링하고, 원격 제어 명령(재부팅, 설치, 메시지 전송 등)을 수행한다.
* **주요 특징:**
    * **단일 바이너리:** 의존성 파일 없이 `.exe` 파일 하나로 실행.
    * **실시간 통신:** WebSocket을 이용한 양방향 실시간 제어.
    * **가벼운 리소스:** 저사양 PC에서도 부담 없이 백그라운드 실행.
    * **Windows 최적화:** 윈도우 서비스 등록 및 시스템 명령어 제어.

---

## 🛠️ 기술 스택 (Tech Stack)

| 구분 | 기술 / 라이브러리 | 설명 |
| :--- | :--- | :--- |
| **Language** | **Go (Golang)** | 1.20+ 버전 권장 |
| **Communication** | **WebSocket** | `github.com/gorilla/websocket` (표준적인 소켓 통신) |
| **Server Framework** | **net/http** | Go 표준 라이브러리 (가볍고 빠름) |
| **Process Control** | **os/exec** | 윈도우 명령어(CMD/PowerShell) 실행 |
| **Windows API** | **golang.org/x/sys** | 윈도우 서비스 등록 및 레지스트리 제어 |
| **Frontend** | **HTML/JS (Vanilla)** | 관리자 대시보드 (외부 프레임워크 최소화) |

---

## 🏗️ 시스템 아키텍처 (Architecture)

```mermaid
graph LR
    subgraph [Admin Dashboard]
        Admin(강사/관리자) -->|Web Browser| Server
    end

    subgraph [Server Side]
        Server[Go Server] 
        DB[(In-Memory/File)]
    end

    subgraph [Client Side - Lab PC 1..N]
        Agent1[Go Agent.exe]
        Agent2[Go Agent.exe]
        AgentN[Go Agent.exe]
    end

    Server <-->|WebSocket (Port:8080)| Agent1
    Server <-->|WebSocket| Agent2
    Server <-->|WebSocket| AgentN
