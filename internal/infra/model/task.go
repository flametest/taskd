package model

import (
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"gorm.io/datatypes"
)

type Task struct {
	Base
	Name       string         `gorm:"column:name"`
	TaskId     string         `gorm:"column:task_id"`
	Protocol   enum.Protocol  `gorm:"column:protocol"`
	Address    string         `gorm:"column:address"`
	Params     datatypes.JSON `gorm:"column:params"`
	ExecTime   time.Time      `gorm:"column:exec_time"`
	Status     enum.Status    `gorm:"column:status"`
	Attempts   int            `gorm:"column:attempts"`
	MaxRetries int            `gorm:"column:max_retries"`
	LastError  string         `gorm:"column:last_error"`
}
