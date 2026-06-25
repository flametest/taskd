package config

import (
	"github.com/flametest/taskd/internal/scheduler"
	"github.com/flametest/vita/vgorm"
	log "github.com/flametest/vita/vlog"
	"github.com/flametest/vita/vserver"
	"github.com/spf13/viper"
)

type Config struct {
	AppConfig  vserver.EchoServerConfig   `json:"app_config" yaml:"AppConfig"`
	LogLevel   log.Level                  `json:"log_level" yaml:"LogLevel"`
	Datasource *vgorm.Config              `json:"datasource" yaml:"Datasource"`
	Scheduler  *scheduler.SchedulerConfig `json:"scheduler" yaml:"Scheduler"`
}

func ParseConfig(path string) (*Config, error) {
	cfg := &Config{}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
