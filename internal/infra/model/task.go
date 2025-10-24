package model

import (
	"time"

	"gorm.io/datatypes"
)

type Task struct {
	Base
	name       string         `gorm:"column:name"`
	taskId     uint64         `gorm:"column:task_id"`
	protocol   string         `gorm:"column:protocol"`
	address    string         `gorm:"column:address"`
	params     datatypes.JSON `gorm:"column:params"`
	execTime   time.Time      `gorm:"column:exec_time"`
	status     string         `gorm:"column:status"`
	attempts   uint64         `gorm:"column:attempts"`
	maxRetries uint64         `gorm:"column:max_retries"`
	lastError  string         `gorm:"column:last_error"`
}
