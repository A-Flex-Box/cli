package config

import (
	"os"
	"path/filepath"

	"github.com/A-Flex-Box/cli/internal/logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	configDir  = "a-flex-box"
	configName = "config"
	configType = "yaml"
)

// Manager loads and saves configuration. No global Viper.
type Manager struct {
	v *viper.Viper
}

// NewManager creates a Manager for ~/.config/a-flex-box/config.yaml.
func NewManager() *Manager {
	dir, _ := os.UserConfigDir()
	cfgDir := filepath.Join(dir, configDir)

	v := viper.New()
	v.SetConfigName(configName)
	v.SetConfigType(configType)
	v.AddConfigPath(cfgDir)
	v.SetDefault("debug", false)
	v.SetDefault("log_level", "info")
	v.SetDefault("wormhole.active_relay", "public")
	v.SetDefault("wormhole.relays", map[string]string{
		"public": "tcp://relay.flex-box.dev:9000",
		"local":  "tcp://127.0.0.1:9000",
	})
	return &Manager{v: v}
}

// Load reads config from disk and unmarshals into Root. Creates default file if missing.
func (m *Manager) Load() (*Root, error) {
	dir, _ := os.UserConfigDir()
	path := filepath.Join(dir, configDir, configName+"."+configType)
	logger.Info("config.Load start", logger.Context("params", map[string]any{
		"path": path, "config_dir": configDir, "config_name": configName,
	})...)
	if err := m.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Info("config file not found, creating default", logger.Context("params", map[string]any{"path": path})...)
			dir, _ := os.UserConfigDir()
			cfgDir := filepath.Join(dir, configDir)
			if err := os.MkdirAll(cfgDir, 0755); err != nil {
				return nil, err
			}
			path := filepath.Join(cfgDir, configName+"."+configType)
			m.v.SetConfigFile(path)
			if err := m.v.WriteConfig(); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var cfg Root
	if err := m.v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	if cfg.Wormhole.Relays == nil {
		cfg.Wormhole.Relays = map[string]string{
			"public": "tcp://relay.flex-box.dev:9000",
			"local":  "tcp://127.0.0.1:9000",
		}
	}
	if cfg.Wormhole.ActiveRelay == "" {
		cfg.Wormhole.ActiveRelay = "public"
	}
	logger.Info("config.Load done", logger.Context("result", map[string]any{
		"debug": cfg.Debug, "log_level": cfg.LogLevel,
		"active_relay": cfg.Wormhole.ActiveRelay, "relays": cfg.Wormhole.Relays,
	})...)
	return &cfg, nil
}

// Save writes the Root config back to disk.
func (m *Manager) Save(cfg *Root) error {
	dir, _ := os.UserConfigDir()
	path := filepath.Join(dir, configDir, configName+"."+configType)
	logger.Info("config.Save start", logger.Context("params", map[string]any{
		"path": path, "debug": cfg.Debug, "log_level": cfg.LogLevel,
		"active_relay": cfg.Wormhole.ActiveRelay, "relays": cfg.Wormhole.Relays,
	})...)
	m.v.Set("debug", cfg.Debug)
	m.v.Set("log_level", cfg.LogLevel)
	m.v.Set("wormhole.active_relay", cfg.Wormhole.ActiveRelay)
	m.v.Set("wormhole.relays", cfg.Wormhole.Relays)
	if err := m.v.WriteConfig(); err != nil {
		logger.Error("config.Save failed", zap.Error(err), zap.String("path", path))
		return err
	}
	logger.Info("config.Save done", zap.String("path", path))
	return nil
}
