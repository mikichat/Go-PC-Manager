package config

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 에이전트 설정 구조체
type Config struct {
	ServerAddress        string `yaml:"server_address"`         // 서버 주소 (예: localhost:8080)
	StatusInterval       int    `yaml:"status_interval"`        // 상태 수집 주기 (초)
	UpdateCheckInterval  int    `yaml:"update_check_interval"`  // 업데이트 확인 주기 (초)
	LogFile              string `yaml:"log_file"`               // 로그 파일 경로
	AuthToken            string `yaml:"auth_token"`             // 인증 토큰 (보안)
}

// DefaultConfig 기본 설정값 반환
func DefaultConfig() *Config {
	return &Config{
		ServerAddress:       "localhost:8080",
		StatusInterval:      5,
		UpdateCheckInterval: 60,
		LogFile:            "agent.log",
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

// GetStatusDuration 상태 수집 주기를 time.Duration으로 반환
func (c *Config) GetStatusDuration() time.Duration {
	return time.Duration(c.StatusInterval) * time.Second
}

// GetUpdateCheckDuration 업데이트 확인 주기를 time.Duration으로 반환
func (c *Config) GetUpdateCheckDuration() time.Duration {
	return time.Duration(c.UpdateCheckInterval) * time.Second
}
