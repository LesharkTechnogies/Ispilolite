package audit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ActorID, ActorRole, Event, Action, ResourceType, ResourceID, Description string
	Success                                                                  bool
	RequestID, IPAddress, UserAgent                                          string
	Metadata                                                                 map[string]interface{}
	OccurredAt                                                               time.Time
}

type Store struct {
	DB   *sql.DB
	Root string
}

var current struct {
	sync.RWMutex
	store *Store
}

func Configure(db *sql.DB, root string) *Store {
	if strings.TrimSpace(root) == "" {
		root = "storage/audit"
	}
	s := &Store{DB: db, Root: root}
	current.Lock()
	current.store = s
	current.Unlock()
	return s
}

func Record(ctx context.Context, event Event) error {
	current.RLock()
	s := current.store
	current.RUnlock()
	if s == nil || s.DB == nil {
		return nil
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	meta, _ := json.Marshal(event.Metadata)
	_, err := s.DB.ExecContext(ctx, `INSERT INTO audit_events(id,actor_id,actor_role,event,action,resource_type,resource_id,description,success,request_id,ip_address,user_agent,metadata,occurred_at) VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, uuid.NewString(), event.ActorID, event.ActorRole, event.Event, event.Action, event.ResourceType, event.ResourceID, event.Description, event.Success, event.RequestID, event.IPAddress, event.UserAgent, meta, event.OccurredAt)
	return err
}

func (s *Store) Export(ctx context.Context, kind string, start, end time.Time, createdBy string) (string, int, error) {
	if !end.After(start) {
		return "", 0, fmt.Errorf("archive end must be after start")
	}
	if err := os.MkdirAll(s.Root, 0750); err != nil {
		return "", 0, err
	}
	name := archiveName(kind, start)
	path := filepath.Join(s.Root, name)
	rows, err := s.DB.QueryContext(ctx, `SELECT id,COALESCE(actor_id::text,''),actor_role,event,action,resource_type,resource_id,description,success,request_id,ip_address,user_agent,metadata,occurred_at FROM audit_events WHERE occurred_at >= $1 AND occurred_at < $2 ORDER BY occurred_at,id`, start, end)
	if err != nil {
		return "", 0, err
	}
	defer rows.Close()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return "", 0, err
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"id", "actor_id", "actor_role", "event", "action", "resource_type", "resource_id", "description", "success", "request_id", "ip_address", "user_agent", "metadata", "occurred_at"})
	count := 0
	for rows.Next() {
		var values [12]string
		var success bool
		var meta []byte
		var at time.Time
		if err := rows.Scan(&values[0], &values[1], &values[2], &values[3], &values[4], &values[5], &values[6], &values[7], &success, &values[8], &values[9], &values[10], &meta, &at); err != nil {
			f.Close()
			return "", count, err
		}
		_ = w.Write([]string{values[0], values[1], values[2], values[3], values[4], values[5], values[6], values[7], strconv.FormatBool(success), values[8], values[9], values[10], string(meta), at.UTC().Format(time.RFC3339Nano)})
		count++
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return "", count, err
	}
	if err := f.Close(); err != nil {
		return "", count, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", count, err
	}
	sum := sha256.Sum256(data)
	_, err = s.DB.ExecContext(ctx, `INSERT INTO audit_archives(id,period_type,period_start,period_end,file_name,storage_path,checksum,event_count,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,'')::uuid) ON CONFLICT(period_type,period_start,period_end) DO UPDATE SET file_name=EXCLUDED.file_name,storage_path=EXCLUDED.storage_path,checksum=EXCLUDED.checksum,event_count=EXCLUDED.event_count,created_at=now()`, uuid.NewString(), strings.ToUpper(kind), start, end, name, path, hex.EncodeToString(sum[:]), count, createdBy)
	return path, count, err
}

func archiveName(kind string, start time.Time) string {
	switch strings.ToUpper(kind) {
	case "DAILY":
		return start.Format("20060102") + "D.csv"
	case "WEEKLY":
		return start.Format("20060102") + "W.csv"
	case "MONTHLY":
		return start.Format("200601") + "M.csv"
	case "YEARLY":
		return start.Format("2006") + "Y.csv"
	default:
		return start.Format("20060102") + "-custom.csv"
	}
}

func (s *Store) DeleteCovered(ctx context.Context, start, end time.Time, event string) (int64, error) {
	var covered bool
	if err := s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM audit_archives WHERE period_start <= $1 AND period_end >= $2 AND checksum<>'')`, start, end).Scan(&covered); err != nil {
		return 0, err
	}
	if !covered {
		return 0, fmt.Errorf("audit deletion requires a completed archive covering the date range")
	}
	res, err := s.DB.ExecContext(ctx, `DELETE FROM audit_events WHERE occurred_at >= $1 AND occurred_at < $2 AND ($3='' OR event=$3)`, start, end, event)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
