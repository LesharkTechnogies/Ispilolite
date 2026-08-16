package middleware

import (
	"context"
	"net/http"
	"net"
	"strconv"
	"strings"
	"time"

	goredis "github.com/go-redis/redis/v8"
)

type RateLimiter struct { client *goredis.Client; limit int64; window time.Duration; prefix string }
func NewRateLimiter(client *goredis.Client,limit int,window time.Duration)*RateLimiter{if limit<=0{limit=60};if window<=0{window=time.Minute};return &RateLimiter{client:client,limit:int64(limit),window:window,prefix:"ratelimit:"}}
var rateScript=goredis.NewScript(`local current=redis.call('INCR',KEYS[1]);if current==1 then redis.call('PEXPIRE',KEYS[1],ARGV[1]) end;local ttl=redis.call('PTTL',KEYS[1]);return {current,ttl}`)
func(r *RateLimiter)Allow(ctx context.Context,key string)(bool,time.Duration,error){if r.client==nil{return true,0,nil};result,err:=rateScript.Run(ctx,r.client,[]string{r.prefix+key},r.window.Milliseconds()).Int64Slice();if err!=nil{return false,0,err};return result[0]<=r.limit,time.Duration(result[1])*time.Millisecond,nil}
func(r *RateLimiter)Middleware(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,req *http.Request){key:=clientIP(req)+":"+routeBucket(req.URL.Path);allowed,retry,err:=r.Allow(req.Context(),key);if err!=nil{http.Error(w,"rate limiter unavailable",http.StatusServiceUnavailable);return};if !allowed{seconds:=int(retry.Seconds());if seconds<1{seconds=1};w.Header().Set("Retry-After",strconv.Itoa(seconds));http.Error(w,"rate limit exceeded",http.StatusTooManyRequests);return};next.ServeHTTP(w,req)})}
func clientIP(r *http.Request)string{if forwarded:=strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"),",")[0]);forwarded!=""{return forwarded};host,_,err:=net.SplitHostPort(r.RemoteAddr);if err==nil{return host};return r.RemoteAddr}
func routeBucket(path string)string{parts:=strings.Split(strings.Trim(path,"/"),"/");if len(parts)>3{return strings.Join(parts[:3],"/")};return strings.Join(parts,"/")}
