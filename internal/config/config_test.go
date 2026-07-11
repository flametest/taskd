package config

import (
	"testing"

	"github.com/flametest/vita/vgorm"
	"github.com/flametest/vita/vserver"
)

func validCfg() *Config {
	return &Config{
		AppConfig: vserver.EchoServerConfig{Name: "taskd", Addr: "0.0.0.0:8080"},
		Datasource: &vgorm.Config{
			Dialect:  vgorm.DialectPostgres,
			Host:     "127.0.0.1",
			Port:     "5432",
			Database: "taskd",
		},
	}
}

func TestConfig_Validate_OK(t *testing.T) {
	if err := validCfg().validate(); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestConfig_Validate_MissingName(t *testing.T) {
	c := validCfg()
	c.AppConfig.Name = ""
	if err := c.validate(); err == nil {
		t.Error("missing name: expected error")
	}
}

func TestConfig_Validate_MissingDatasource(t *testing.T) {
	c := validCfg()
	c.Datasource = nil
	if err := c.validate(); err == nil {
		t.Error("nil datasource: expected error")
	}
}

func TestConfig_Validate_BadDialect(t *testing.T) {
	c := validCfg()
	c.Datasource.Dialect = "oracle"
	if err := c.validate(); err == nil {
		t.Error("bad dialect: expected error")
	}
}

func TestConfig_Validate_MissingHostForPostgres(t *testing.T) {
	c := validCfg()
	c.Datasource.Host = ""
	if err := c.validate(); err == nil {
		t.Error("missing host for postgres: expected error")
	}
}

// sqlite3 uses Database as the file path, so host/port are not required.
func TestConfig_Validate_SQLiteWithoutHost(t *testing.T) {
	c := validCfg()
	c.Datasource.Dialect = vgorm.DialectSQLite3
	c.Datasource.Host = ""
	c.Datasource.Port = ""
	c.Datasource.Database = ":memory:"
	if err := c.validate(); err != nil {
		t.Errorf("sqlite3 without host: %v", err)
	}
}
