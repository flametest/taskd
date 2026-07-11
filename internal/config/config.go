package config

import (
	"errors"
	"fmt"

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
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

// validate checks required fields at startup so the process fails fast with a
// clear message instead of crashing later at runtime - e.g. a nil datasource, an
// unsupported dialect that vgorm.NewDialector would panic on, or a missing DB
// host. Scheduler fields are intentionally not validated here: zero values are
// given sane defaults by scheduler.ResolveSchedulerConfig.
func (c *Config) validate() error {
	if c.AppConfig.Name == "" {
		return errors.New("app_config.name is required")
	}
	if c.AppConfig.Addr == "" {
		return errors.New("app_config.addr is required")
	}
	if c.Datasource == nil {
		return errors.New("datasource is required")
	}
	switch c.Datasource.Dialect {
	case vgorm.DialectPostgres, vgorm.DialectMySQL, vgorm.DialectSQLite3:
	default:
		return fmt.Errorf("datasource.dialect %q is not supported (use postgres, mysql, or sqlite3)", c.Datasource.Dialect)
	}
	if c.Datasource.Database == "" {
		return errors.New("datasource.database is required")
	}
	// postgres/mysql need a host:port; sqlite3 uses Database as the file path.
	if c.Datasource.Dialect != vgorm.DialectSQLite3 {
		if c.Datasource.Host == "" {
			return errors.New("datasource.host is required for postgres/mysql")
		}
		if c.Datasource.Port == "" {
			return errors.New("datasource.port is required for postgres/mysql")
		}
	}
	return nil
}
