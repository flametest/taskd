package model

import (
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/vita/vgorm"
	"gorm.io/datatypes"
)

type Task struct {
	vgorm.BasePostgres
	Name        string         `gorm:"column:name"`
	RefId       string         `gorm:"column:ref_id"`
	Protocol    enum.Protocol  `gorm:"column:protocol"`
	Address     string         `gorm:"column:address"`
	Params      datatypes.JSON `gorm:"column:params"`
	ExecTime    time.Time      `gorm:"column:exec_time"`
	Status      enum.Status    `gorm:"column:status"`
	Attempts    int            `gorm:"column:attempts"`
	MaxRetries  int            `gorm:"column:max_retries"`
	LastError   string         `gorm:"column:last_error"`
	LockedUntil *time.Time     `gorm:"column:locked_until"`
}
