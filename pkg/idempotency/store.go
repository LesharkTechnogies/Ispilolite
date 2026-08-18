package idempotency

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Store struct{ DB *sql.DB }

type Result struct {
	Claimed    bool
	StatusCode int
	Response   json.RawMessage
}

func HashRequest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (s *Store) Claim(userID, scope, key, requestHash string) (Result, error) {
	if s == nil || s.DB == nil {
		return Result{}, fmt.Errorf("idempotency store is not configured")
	}
	if strings.TrimSpace(key) == "" {
		return Result{}, fmt.Errorf("Idempotency-Key header is required")
	}
	var status sql.NullInt64
	var response []byte
	err := s.DB.QueryRow(`INSERT INTO idempotency_keys(user_id,scope,key,request_hash) VALUES($1,$2,$3,$4) ON CONFLICT(user_id,scope,key) DO UPDATE SET key=idempotency_keys.key RETURNING status_code,response`, userID, scope, key, requestHash).Scan(&status, &response)
	if err == nil && !status.Valid {
		return Result{Claimed: true}, nil
	}
	if err != nil {
		return Result{}, err
	}
	var existingHash string
	if err := s.DB.QueryRow(`SELECT request_hash FROM idempotency_keys WHERE user_id=$1 AND scope=$2 AND key=$3`, userID, scope, key).Scan(&existingHash); err != nil {
		return Result{}, err
	}
	if existingHash != requestHash {
		return Result{}, fmt.Errorf("idempotency key was reused with a different request")
	}
	return Result{Claimed: false, StatusCode: int(status.Int64), Response: response}, nil
}

func (s *Store) Complete(userID, scope, key string, statusCode int, response any) error {
	body, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`UPDATE idempotency_keys SET status_code=$1,response=$2,completed_at=$3 WHERE user_id=$4 AND scope=$5 AND key=$6`, statusCode, body, time.Now().UTC(), userID, scope, key)
	return err
}
