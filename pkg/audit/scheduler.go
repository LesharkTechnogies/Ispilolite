package audit

import (
	"context"
	"log"
	"time"
)

// StartScheduler archives completed periods at 03:00 in the supplied location.
// It runs once at startup when the process starts during the archive window.
func (s *Store) StartScheduler(ctx context.Context, location *time.Location, logger *log.Logger) {
	if location == nil {
		location = time.Local
	}
	if logger == nil {
		logger = log.Default()
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				local := now.In(location)
				if local.Hour() != 3 || local.Minute() != 0 {
					continue
				}
				dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
				previousDay := dayStart.AddDate(0, 0, -1)
				if _, count, err := s.Export(ctx, "DAILY", previousDay, dayStart, ""); err != nil {
					logger.Printf("audit daily archive failed: %v", err)
				} else {
					logger.Printf("audit daily archive complete: %d events", count)
				}
				if local.Weekday() == time.Monday {
					start := previousDay.AddDate(0, 0, -6)
					if _, count, err := s.Export(ctx, "WEEKLY", start, dayStart, ""); err != nil {
						logger.Printf("audit weekly archive failed: %v", err)
					} else {
						logger.Printf("audit weekly archive complete: %d events", count)
					}
				}
				if local.Day() == 1 {
					start := dayStart.AddDate(0, -1, 0)
					if _, count, err := s.Export(ctx, "MONTHLY", start, dayStart, ""); err != nil {
						logger.Printf("audit monthly archive failed: %v", err)
					} else {
						logger.Printf("audit monthly archive complete: %d events", count)
					}
				}
				if local.Month() == time.January && local.Day() == 1 {
					start := dayStart.AddDate(-1, 0, 0)
					if _, count, err := s.Export(ctx, "YEARLY", start, dayStart, ""); err != nil {
						logger.Printf("audit yearly archive failed: %v", err)
					} else {
						logger.Printf("audit yearly archive complete: %d events", count)
					}
				}
			}
		}
	}()
}
