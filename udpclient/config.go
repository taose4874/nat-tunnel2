package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
	"math/rand"
)

type Config struct {
	PunchHole PunchHoleConfig `json:"punch_hole"`
	Server    ServerConfig    `json:"server"`
	LogLevel  string          `json:"log_level"`
	TunIP     string          `json:"tun_ip"`
	ClientID  string          `json:"client_id"`
	ClientPwd string          `json:"client_pwd"`
}

type PunchHoleConfig struct {
	MaxConcurrency int  `json:"max_concurrency"`
	PortRange      int  `json:"port_range"`
	BasePortOffset int  `json:"base_port_offset"`
	EnableRelay    bool `json:"enable_relay"`
	PunchTimeout   int  `json:"punch_timeout"`
	RelayFallback  bool `json:"relay_fallback"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

var (
	config     *Config
	configOnce sync.Once
	configPath string
)

func GetConfig() *Config {
	configOnce.Do(func() {
		configPath = filepath.Join(getConfigDir(), "config.json")
		config = loadConfig()
	})
	return config
}

func loadConfig() *Config {
	defaultConfig := &Config{
		PunchHole: PunchHoleConfig{
			MaxConcurrency: 2,
			PortRange:      16,
			BasePortOffset: 0,
			EnableRelay:    true,
			PunchTimeout:   20,
			RelayFallback:  true,
		},
		Server: ServerConfig{
			Host: "117.72.206.26",
			Port: 17709,
		},
		LogLevel:  "INFO",
		TunIP:     "10.10.10.199",
		ClientID:  generateClientID(),
		ClientPwd: generatePassword(),
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		saveConfig(defaultConfig)
		return defaultConfig
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		saveConfig(defaultConfig)
		return defaultConfig
	}

	loadedConfig := &Config{}
	if err := json.Unmarshal(data, loadedConfig); err != nil {
		saveConfig(defaultConfig)
		return defaultConfig
	}

	if loadedConfig.ClientID == "" {
		loadedConfig.ClientID = defaultConfig.ClientID
	}
	if loadedConfig.ClientPwd == "" {
		loadedConfig.ClientPwd = defaultConfig.ClientPwd
	}

	return loadedConfig
}

func saveConfig(cfg *Config) error {
	os.MkdirAll(getConfigDir(), 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func (c *Config) Save() error {
	return saveConfig(c)
}

func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./config"
	}
	return filepath.Join(home, ".natun")
}

func generateClientID() string {
	const chars = "0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := make([]byte, 8)
	for i := range id {
		id[i] = chars[r.Intn(10)]
	}
	return string(id)
}

func generatePassword() string {
	const chars = "0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pwd := make([]byte, 6)
	for i := range pwd {
		pwd[i] = chars[r.Intn(10)]
	}
	return string(pwd)
}
