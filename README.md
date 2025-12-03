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
    * **에이전트 식별:** 호스트명과 MAC 주소 기반 고유 ID로 각 PC 구분.
    * **실시간 모니터링:** CPU, 메모리, 디스크 사용률 및 업타임 실시간 수집.
    * **명령 결과 반환:** 원격 명령 실행 결과를 대시보드에서 확인 가능.
    * **자동 재연결:** 네트워크 오류 시 자동으로 서버에 재연결.

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
```

---

## ✨ 주요 기능 (Features)

### 1. 에이전트 관리
- **고유 식별:** 호스트명과 MAC 주소를 조합한 고유 ID로 각 PC 식별
- **자동 등록:** 에이전트 실행 시 서버에 자동 등록
- **연결 상태 모니터링:** 실시간으로 에이전트 연결 상태 확인

### 2. 시스템 모니터링
- **CPU 사용률:** 실시간 CPU 사용률 수집 및 표시
- **메모리 사용률:** 시스템 메모리 사용률 모니터링
- **디스크 사용률:** C: 드라이브 디스크 사용률 확인
- **업타임:** 에이전트 실행 시간 추적

### 3. 원격 명령 실행
- **전체 브로드캐스트:** 모든 연결된 에이전트에 동시 명령 전송
- **개별 전송:** 특정 에이전트에만 명령 전송
- **스크린샷 캡처:** 원격 PC의 현재 화면을 실시간으로 캡처하여 확인
- **결과 확인:** 명령 실행 결과를 대시보드에서 실시간 확인
- **에러 처리:** 명령 실행 실패 시 상세한 에러 정보 제공

### 4. 웹 대시보드
- **에이전트 목록:** 연결된 모든 에이전트 정보 표시
- **상태 정보:** 각 에이전트의 실시간 시스템 상태 표시
- **명령 전송:** 웹 인터페이스를 통한 쉬운 명령 전송
- **결과 표시:** 명령 실행 결과를 깔끔한 UI로 확인

### 5. 안정성
- **자동 재연결:** 네트워크 오류 시 자동으로 서버에 재연결
- **에러 로깅:** 상세한 에러 로그로 문제 추적 용이
- **연결 복구:** 일시적인 연결 끊김 시 자동 복구

---

## 🚀 사용 방법 (Usage)

### 서버 실행
```bash
cd server
go run main.go
# 또는 빌드 후 실행
go build -o gopc-server.exe
./gopc-server.exe
```

서버는 기본적으로 `http://localhost:8080`에서 실행됩니다.

### 에이전트 실행 (Agent Execution)

GUI 프로그램(메모장 등) 실행을 위해 에이전트는 **사용자 모드**에서 실행되어야 합니다. 이를 위해 간편한 배치 스크립트를 제공합니다.

#### 1. 수동 실행
제공된 배치 파일을 사용하여 에이전트를 빌드하고 실행합니다.
```bash
cd agent
run_agent.bat
```

#### 2. 윈도우 시작 프로그램 등록 (자동 실행)
PC 로그인 시 에이전트가 자동으로 실행되도록 시작 프로그램에 등록합니다.
```bash
cd agent
install_startup.bat
```
* 등록을 해제하려면 `uninstall_startup.bat`를 실행하세요.

에이전트는 자동으로 서버에 연결을 시도하며, 연결이 끊어지면 자동으로 재연결합니다.

### 대시보드 접속
웹 브라우저에서 `http://localhost:8080`에 접속하여 관리 대시보드를 사용할 수 있습니다.

---

## 📡 메시지 프로토콜 (Message Protocol)

시스템은 JSON 기반 메시지 프로토콜을 사용합니다.

### 메시지 타입
- `register`: 에이전트 등록
- `command`: 명령 전송
- `command_result`: 명령 실행 결과
- `status`: 상태 정보
- `agent_list`: 에이전트 목록
- `agent_update`: 에이전트 정보 업데이트

### 예시
```json
{
  "type": "command",
  "agent_id": "PC-01-00:11:22:33:44:55",
  "command": "dir"
}
```

---

## 🔧 설정 (Configuration)

### 설정 파일 사용 (권장)

에이전트와 서버는 YAML 형식의 설정 파일을 지원합니다. 설정 파일을 사용하면 재컴파일 없이 쉽게 설정을 변경할 수 있습니다.

#### 에이전트 설정

1. 예시 파일을 복사하여 설정 파일 생성:
```bash
cd agent
copy config.example.yaml config.yaml
```

2. `config.yaml` 파일 수정:
```yaml
# 서버 주소 (호스트:포트)
server_address: "your-server-ip:8080"

# 상태 정보 수집 주기 (초)
status_interval: 5

# 업데이트 확인 주기 (초)
update_check_interval: 60

# 로그 파일 경로
log_file: "agent.log"
```

#### 서버 설정

1. 예시 파일을 복사하여 설정 파일 생성:
```bash
cd server
copy config.example.yaml config.yaml
```

2. `config.yaml` 파일 수정:
```yaml
# 서버 포트
port: "8080"

# 정적 파일 디렉토리
static_dir: "static"

# 업데이트 파일 디렉토리
updates_dir: "updates"

# 현재 에이전트 버전
agent_version: "1.0.1"
```

### 기본값

설정 파일이 없을 경우 다음 기본값으로 동작합니다:

**에이전트:**
- 서버 주소: `localhost:8080`
- 상태 수집 주기: `5초`
- 업데이트 확인 주기: `60초`

**서버:**
- 포트: `8080`
- 정적 파일 디렉토리: `static`
- 업데이트 파일 디렉토리: `updates`

### 코드에서 직접 변경 (비권장)

설정 파일을 사용하지 않고 코드에서 직접 변경하려면:

**에이전트 서버 주소 변경:**
`agent/config/config.go`의 `DefaultConfig()` 함수 수정

**서버 포트 변경:**
`server/config/config.go`의 `DefaultConfig()` 함수 수정

---

## 📝 추가 개선 사항 (Future Improvements)

다음 기능들은 향후 개선을 위해 계획된 항목입니다:

### 보안 강화
- [ ] TLS/SSL 지원 (HTTPS/WSS)
- [ ] 인증 및 권한 관리 시스템
- [ ] API 키 기반 접근 제어
- [ ] CORS 설정 개선 (현재는 모든 Origin 허용)

### 기능 확장
- [ ] 파일 전송 기능 (에이전트 간 파일 공유)
- [x] 스크린샷 캡처 기능
- [ ] 원격 데스크톱 제어
- [ ] 프로그램 자동 설치/제거 기능
- [ ] 스케줄링된 명령 실행 (cron-like)
- [ ] 명령 히스토리 저장 및 조회

### 모니터링 개선
- [ ] 네트워크 사용량 모니터링
- [ ] 실행 중인 프로세스 목록 조회
- [ ] 설치된 프로그램 목록 조회
- [ ] 이벤트 로그 수집 (Windows Event Log)
- [ ] 알림 시스템 (임계값 초과 시 알림)

### 데이터 관리
- [ ] 데이터베이스 연동 (SQLite/PostgreSQL)
- [ ] 상태 정보 히스토리 저장
- [ ] 통계 및 리포트 생성
- [ ] 데이터 백업 및 복원

### 사용자 경험
- [ ] 다크 모드 지원
- [ ] 다국어 지원 (영어, 일본어 등)
- [ ] 반응형 디자인 개선
- [ ] 키보드 단축키 지원
- [ ] 명령 템플릿 기능

### 운영 편의성
- [ ] Windows 서비스로 등록 기능
- [x] 설정 파일 지원 (YAML/JSON)
- [ ] 로그 파일 로테이션
- [ ] 원격 업데이트 기능
- [ ] 에이전트 그룹 관리

### 성능 최적화
- [ ] 메시지 압축 (gzip)
- [ ] 연결 풀링
- [ ] 상태 정보 캐싱
- [ ] 대량 명령 실행 최적화

### 플랫폼 확장
- [ ] Linux 에이전트 지원 (현재는 Windows만 완전 지원)
- [ ] macOS 에이전트 지원
- [ ] 모바일 앱 (관리자용)

---

## 🐛 알려진 이슈 (Known Issues)

- Windows에서만 완전한 시스템 정보 수집이 가능 (Linux/Mac는 기본값 반환)
- CORS 체크가 모든 Origin을 허용하도록 설정됨 (개발 환경용)
- 대량의 에이전트 연결 시 성능 최적화 필요

---

## 📄 라이선스 (License)

이 프로젝트는 교육 목적으로 제작되었습니다.
