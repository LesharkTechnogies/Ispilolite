package redis

import (
	"context"
	"encoding/json"
	"time"

	goredis "github.com/go-redis/redis/v8"
	"ispilolite/pkg/database"
)

type Session struct { UserID string `json:"user_id"`; Role string `json:"role"`; DeviceID string `json:"device_id,omitempty"`; CreatedAt time.Time `json:"created_at"` }
type SessionRepository struct { client *goredis.Client }
func NewSessionRepository()*SessionRepository{return &SessionRepository{client:database.GetRedis()}}
func(r *SessionRepository)Set(ctx context.Context,id string,session Session,ttl time.Duration)error{body,err:=json.Marshal(session);if err!=nil{return err};return r.client.Set(ctx,"session:"+id,body,ttl).Err()}
func(r *SessionRepository)Get(ctx context.Context,id string)(*Session,error){body,err:=r.client.Get(ctx,"session:"+id).Bytes();if err!=nil{return nil,err};session:=&Session{};if err:=json.Unmarshal(body,session);err!=nil{return nil,err};return session,nil}
func(r *SessionRepository)Delete(ctx context.Context,id string)error{return r.client.Del(ctx,"session:"+id).Err()}
func(r *SessionRepository)Touch(ctx context.Context,id string,ttl time.Duration)error{return r.client.Expire(ctx,"session:"+id,ttl).Err()}
