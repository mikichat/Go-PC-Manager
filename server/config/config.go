package config

import (
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 서버 설정 구조체
type Config struct {
	Port         string `yaml:"port"`          // 서버 포트 (예: 8080)
	StaticDir    string `yaml:"static_dir"`    // 정적 파일 디렉토리
	UpdatesDir   string `yaml:"updates_dir"`   // 업데이트 파일 디렉토리
	AgentVersion string `yaml:"agent_version"` // 현재 에이전트 버전
	AuthToken    string `yaml:"auth_token"`    // 인증 토큰 (보안)
}

// DefaultConfig 기본 설정값 반환
func DefaultConfig() *Config {
	return &Config{
		Port:         "8080",
		StaticDir:    "static",
		UpdatesDir:   "updates",
		AgentVersion: "1.0.1",
	}
}

// Load 설정 파일 로드 (파일이 없으면 기본값 사용)
func Load() *Config {
	cfg := DefaultConfig()

	// 실행 파일 경로 가져오기
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("설정: 실행 파일 경로를 가져올 수 없습니다. 기본값 사용: %v", err)
		return cfg
	}

	// 설정 파일 경로
	configPath := filepath.Join(filepath.Dir(exePath), "config.yaml")

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("설정: config.yaml 파일이 없습니다. 기본값 사용")
		} else {
			log.Printf("설정: 파일 읽기 오류. 기본값 사용: %v", err)
		}
		return cfg
	}

	// YAML 파싱
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Printf("설정: YAML 파싱 오류. 기본값 사용: %v", err)
		return cfg
	}

	log.Printf("설정: config.yaml 로드 완료")
	return cfg
}

// GetListenAddr 서버 리스닝 주소 반환
func (c *Config) GetListenAddr() string {
	return ":" + c.Port
}
