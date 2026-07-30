// Package converter holds domain -> DTO mappings used by the API handler layer.
// The service layer returns domain types; conversion to the wire-format DTO
// happens here so the service stays free of presentation concerns.
package converter

import (
	"github.com/flametest/taskd/internal/domain"
	"github.com/flametest/taskd/pkg/dto"
)

// ToTask maps a domain.Task to its API DTO. It exposes Status (which domain.Task
// keeps private, accessed via Status()) and stringifies LastError (an error in
// the domain type).
func ToTask(t *domain.Task) *dto.Task {
	var lastError string
	if t.LastError != nil {
		lastError = t.LastError.Error()
	}
	return &dto.Task{
		Id:          t.Id,
		Name:        t.Name,
		RefId:       t.RefId,
		Protocol:    t.Protocol,
		Address:     t.Address,
		Params:      t.Params,
		ExecTime:    t.ExecTime,
		Status:      t.Status(),
		Attempts:    t.Attempts,
		MaxRetries:  t.MaxRetries,
		LastError:   lastError,
		LockedUntil: t.LockedUntil,
		Cron:        t.Cron,
	}
}

// ToTasks maps a slice of domain.Task to DTOs.
func ToTasks(tasks []*domain.Task) []*dto.Task {
	out := make([]*dto.Task, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, ToTask(t))
	}
	return out
}
