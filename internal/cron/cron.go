// Package cron parses cron expressions (standard 5-field and @every/@daily
// descriptors) and computes the next activation time in a given timezone. It is
// shared by the service layer (create-time validation) and the scheduler
// (rescheduling after a successful recurring execution), so neither layer needs
// to import the other.
package cron

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// parser supports both standard 5-field cron and descriptors (@every, @daily,
// ...). This is the library's own standardParser flag set. Parser is a value
// type with no mutable state, safe to share across goroutines.
var parser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// cache memoizes parsed schedules keyed by expression. cron.Schedule impls are
// read-only after Parse, so concurrent Next calls on a cached schedule are safe.
var cache sync.Map // string -> cron.Schedule

// Validate reports whether expr is a supported cron expression. It rejects specs
// with a TZ=/CRON_TZ= prefix (which would silently override the configured
// global timezone) and unparseable specs. The parsed schedule is cached on
// success.
func Validate(expr string) error {
	_, err := parse(expr)
	return err
}

// Next returns the next activation time strictly after from, evaluated in loc.
// expr is parsed and cached on first use. The returned time may be zero for an
// unsatisfiable spec (e.g. "0 9 31 2 *"); callers must guard against that.
func Next(expr string, from time.Time, loc *time.Location) (time.Time, error) {
	sched, err := parse(expr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(from.In(loc)), nil
}

func parse(expr string) (cron.Schedule, error) {
	if s, ok := cache.Load(expr); ok {
		return s.(cron.Schedule), nil
	}
	if hasTZPrefix(expr) {
		return nil, fmt.Errorf("TZ=/CRON_TZ= prefix is not allowed, configure the global timezone instead")
	}
	sched, err := parser.Parse(expr)
	if err != nil {
		return nil, err
	}
	cache.Store(expr, sched)
	return sched, nil
}

// hasTZPrefix reports whether expr embeds a per-expression TZ=/CRON_TZ= timezone
// prefix, which would silently override the configured global timezone.
func hasTZPrefix(expr string) bool {
	trimmed := strings.TrimSpace(expr)
	return strings.HasPrefix(trimmed, "TZ=") || strings.HasPrefix(trimmed, "CRON_TZ=")
}
