package model

import "time"

type Base struct {
	Id        uint64    `gorm:"primaryKey;autoIncrement;column:id"`
	Version   uint64    `gorm:"column:version"`
	CreatedAt time.Time `gorm:"<-:create;index;type:DATETIME;default:CURRENT_TIMESTAMP not null;column:created_at"`
	UpdatedAt time.Time `gorm:"type:DATETIME;default:CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP;column:updated_at"`
}
