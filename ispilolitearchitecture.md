Comprehensive System Design & Architecture Document
"Ispilo Lite" - ISP Discovery & Installation Platform
Version: 2.0 (High-Scale Architecture)
Target: 10,000+ Concurrent Requests
Date: October 26, 2023

1. Executive Summary
This document outlines the production-grade architecture for Ispilo Lite, a multi-sided marketplace connecting customers, ISPs, and technicians. The system is designed to handle 10,000 concurrent requests (API calls, WebSocket connections, and background jobs) with high availability, data redundancy, and geospatial intelligence at its core.

2. High-Level Architecture Overview
2.1. System Diagram
text
┌─────────────────────────────────────────────────────────────────────┐
│                         CDN / CloudFront                           │
│                    (Static Assets, Images, PDFs)                   │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                      Load Balancer (HAProxy/Nginx)                 │
│                   (SSL Termination, Rate Limiting)                  │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                    API Gateway (Kong/Traefik)                      │
│        (Routing, Authentication, Caching, Circuit Breakers)         │
└───────┬─────────────────┬─────────────────┬─────────────────────────┘
        │                 │                 │
┌───────▼──────┐ ┌────────▼────────┐ ┌─────▼──────────┐
│  Go API      │ │  Go API         │ │  Go API        │
│  Service 1   │ │  Service 2      │ │  Service 3     │
│  (Core)      │ │  (Matching)     │ │  (Notifications)│
│  - Auth      │ │  - Requests     │ │  - Push        │
│  - Users     │ │  - Quotations   │ │  - SMS         │
│  - Reviews   │ │  - Jobs         │ │  - Email       │
└───────┬──────┘ └────────┬────────┘ └─────┬──────────┘
        │                 │                 │
┌───────▼─────────────────▼─────────────────▼─────────────────────────┐
│                        Message Queue (RabbitMQ/Kafka)              │
│              (Async Jobs, Event-Driven Processing)                  │
└─────────────────────────────────────────────────────────────────────┘
        │                 │                 │
┌───────▼─────────────────▼─────────────────▼─────────────────────────┐
│                          Redis Cluster                              │
│     (Caching, Session Storage, Rate Limiting, WebSocket State)      │
└─────────────────────────────────────────────────────────────────────┘
        │                 │                 │
┌───────▼─────────────────▼─────────────────▼─────────────────────────┐
│                    PostgreSQL Cluster (Primary + Replicas)         │
│                                                                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐    │
│  │   Primary DB    │  │  Replica 1      │  │  Replica 2      │    │
│  │   (Write)       │  │  (Read/Geospatial│  │  (Read/Reports) │    │
│  │   + PostGIS     │  │   Queries)      │  │                 │    │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘    │
│                                                                     │
│                 PostGIS Extension (Geospatial Indexing)             │
└─────────────────────────────────────────────────────────────────────┘
        │                 │                 │
┌───────▼─────────────────▼─────────────────▼─────────────────────────┐
│                    Elasticsearch Cluster                            │
│           (Search, Fuzzy Matching, Analytics)                       │
└─────────────────────────────────────────────────────────────────────┘
        │                 │                 │
┌───────▼─────────────────▼─────────────────▼─────────────────────────┐
│                    S3 Compatible Storage                            │
│            (Images, Watermarked Photos, PDF Quotes)                 │
└─────────────────────────────────────────────────────────────────────┘
3. Database Architecture (PostgreSQL + PostGIS + Replicas)
3.1. Database Cluster Setup
Component	Purpose	Specifications
Primary DB	All write operations (INSERT, UPDATE, DELETE).	16 vCPU, 64GB RAM, 1TB SSD
Replica 1 (Geospatial)	Dedicated for ST_DWithin, ST_Contains, and radius queries.	16 vCPU, 64GB RAM, 1TB SSD
Replica 2 (Reporting)	Analytics, heatmaps, admin dashboards.	8 vCPU, 32GB RAM, 2TB SSD
Read Replica 3 (Fallback)	Disaster recovery, backup.	8 vCPU, 32GB RAM, 1TB SSD
3.2. Connection Pooling
go
// Using pgxpool with connection routing
type DBConnectionPool struct {
    WritePool  *pgxpool.Pool  // Primary
    ReadPool   *pgxpool.Pool  // Replica 1 (Geospatial)
    ReportPool *pgxpool.Pool  // Replica 2 (Analytics)
}

func (p *DBConnectionPool) GetConn(operation string) *pgxpool.Pool {
    switch operation {
    case "write", "transaction":
        return p.WritePool
    case "geo", "nearby":
        return p.ReadPool
    case "analytics", "heatmap":
        return p.ReportPool
    default:
        return p.ReadPool
    }
}
Pool Sizes:

Write Pool: 200 connections

Read Pool: 300 connections

Report Pool: 100 connections

3.3. PostGIS Optimization for 10K Requests
Critical Indexes
sql
-- Geospatial GiST Index (For radius searches)
CREATE INDEX idx_locations_coord_gist 
ON locations USING GIST (coord);

-- Combined with BTREE for filtering
CREATE INDEX idx_locations_coord_status 
ON locations USING GIST (coord, status);  -- PostGIS 3.0+ supports multi-column GiST

-- Coverage polygon index
CREATE INDEX idx_isp_coverages_polygon 
ON isp_coverages USING GIST (coverage_area);

-- For popularity ranking
CREATE INDEX idx_locations_popularity 
ON locations (popularity_score DESC);

-- For duplicate detection (fuzzy matching)
CREATE INDEX idx_locations_name_trgm 
ON locations USING GIN (name gin_trgm_ops);
Geospatial Query Optimization
sql
-- Optimized radius search with filtering
SELECT 
    u.id, u.name, u.rating,
    ST_Distance(l.coord, ST_SetSRID(ST_MakePoint($1, $2), 4326)) AS distance
FROM locations l
JOIN users u ON u.id = l.user_id
WHERE 
    ST_DWithin(
        l.coord, 
        ST_SetSRID(ST_MakePoint($1, $2), 4326), 
        $3  -- radius in meters
    )
    AND l.is_verified = true
    AND u.is_active = true
ORDER BY distance ASC
LIMIT 50;
3.4. Partitioning Strategy
To handle 10K requests, implement table partitioning for high-volume tables:

sql
-- Partition jobs by month
CREATE TABLE jobs_2024_01 PARTITION OF jobs
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE jobs_2024_02 PARTITION OF jobs
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Partition notifications by week
CREATE TABLE notifications_2024_w1 PARTITION OF notifications
FOR VALUES FROM ('2024-01-01') TO ('2024-01-08');
3.5. Database Replication Strategy
yaml
# PostgreSQL Replication Configuration
replication:
  mode: synchronous_commit
  sync_standby: 1  # At least 1 replica confirms writes
  wal_keep_segments: 1000
  max_wal_senders: 10
  
  # For read replicas
  hot_standby: on
  max_standby_streaming_delay: 30s
  wal_receiver_status_interval: 10s
4. Go Backend Architecture (Microservices)
4.1. Service Breakdown
Service	Responsibilities	Port	Scaling
Auth Service	JWT creation, OTP, user management	8001	5 replicas
Core Service	CRUD operations, profiles, reviews	8002	10 replicas
Geospatial Service	Radius search, coverage mapping, location intelligence	8003	8 replicas
Matching Service	Request matching, ISP/technician assignment	8004	6 replicas
Quotation Service	Quote generation, PDF creation	8005	4 replicas
Notification Service	Push, SMS, Email, WebSocket	8006	3 replicas
Chat Service	Real-time messaging	8007	4 replicas (WebSocket)
File Service	Upload, watermarking, S3 storage	8008	3 replicas
4.2. Go Code Structure
text
backend/
├── cmd/
│   ├── auth/
│   ├── core/
│   ├── geospatial/
│   └── matching/
├── internal/
│   ├── models/
│   │   ├── user.go
│   │   ├── location.go
│   │   ├── job.go
│   │   └── quotation.go
│   ├── repository/
│   │   ├── postgres/
│   │   │   ├── user_repo.go
│   │   │   ├── location_repo.go
│   │   │   └── job_repo.go
│   │   ├── redis/
│   │   │   ├── cache.go
│   │   │   └── session.go
│   │   └── elasticsearch/
│   │       └── search_repo.go
│   ├── services/
│   │   ├── auth/
│   │   ├── matching/
│   │   ├── geospatial/
│   │   └── notification/
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── ratelimit.go
│   │   ├── logger.go
│   │   └── circuit_breaker.go
│   └── utils/
│       ├── geoutils.go
│       ├── watermark.go
│       └── pdf_generator.go
├── pkg/
│   ├── database/
│   │   ├── postgres.go
│   │   └── redis.go
│   ├── queue/
│   │   └── rabbitmq.go
│   └── monitoring/
│       ├── prometheus.go
│       └── tracing.go
├── api/
│   ├── handlers/
│   ├── dto/
│   └── routes/
├── config/
│   └── config.yaml
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yaml
├── go.mod
└── go.sum
4.3. High-Performance Go Patterns
Connection Pooling with pgxpool
go
package database

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "sync"
)

type DBManager struct {
    WritePool  *pgxpool.Pool
    ReadPool   *pgxpool.Pool
    ReportPool *pgxpool.Pool
    mu         sync.RWMutex
}

func NewDBManager(config *Config) (*DBManager, error) {
    // Write pool (Primary)
    writePool, err := pgxpool.New(context.Background(), config.WriteDSN)
    if err != nil {
        return nil, err
    }
    writePool.Config().MaxConns = 200
    
    // Read pool (Replica)
    readPool, err := pgxpool.New(context.Background(), config.ReadDSN)
    if err != nil {
        return nil, err
    }
    readPool.Config().MaxConns = 300
    
    return &DBManager{
        WritePool: writePool,
        ReadPool:  readPool,
    }, nil
}
Async Processing with RabbitMQ
go
package queue

import (
    "github.com/rabbitmq/amqp091-go"
    "encoding/json"
)

type QueueService struct {
    conn *amqp091.Connection
    ch   *amqp091.Channel
}

type JobEvent struct {
    JobID      string    `json:"job_id"`
    Action     string    `json:"action"` // "assign", "start", "complete"
    Technician string    `json:"technician_id"`
    Timestamp  time.Time `json:"timestamp"`
}

func (q *QueueService) PublishJobEvent(event JobEvent) error {
    body, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    return q.ch.Publish(
        "job_exchange",     // exchange
        "job.routing_key",  // routing key
        false,              // mandatory
        false,              // immediate
        amqp091.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    )
}
5. Caching Strategy (Redis)
5.1. Redis Cluster Setup
yaml
redis:
  cluster:
    nodes: 6  # 3 master + 3 replica
    slots: 16384
    maxmemory: 32GB
    eviction: allkeys-lru
5.2. Cache Layers
Cache Type	Purpose	TTL	Key Pattern
User Session	JWT refresh tokens, user context	24h	session:{user_id}
Geospatial Cache	Nearby ISP results	5min	geo:nearby:{lat}:{lng}:{radius}
Popular Locations	Top searched villages	1hr	locations:popular
Quote Cache	Generated PDF quotes	7d	quote:{quote_id}
Rate Limiting	API request counters	1min	ratelimit:{user_id}:{endpoint}
WebSocket State	User connections	10min	ws:{user_id}
5.3. Cache Implementation
go
package cache

import (
    "context"
    "encoding/json"
    "github.com/redis/go-redis/v9"
    "time"
)

type CacheService struct {
    client *redis.ClusterClient
}

func (c *CacheService) GetNearbyISPs(ctx context.Context, lat, lng float64, radius int) ([]ISP, error) {
    key := fmt.Sprintf("geo:nearby:%.4f:%.4f:%d", lat, lng, radius)
    
    // Try cache first
    cached, err := c.client.Get(ctx, key).Result()
    if err == nil {
        var isps []ISP
        json.Unmarshal([]byte(cached), &isps)
        return isps, nil
    }
    
    // Cache miss - query database
    isps, err := c.queryDB(ctx, lat, lng, radius)
    if err != nil {
        return nil, err
    }
    
    // Store in cache
    data, _ := json.Marshal(isps)
    c.client.Set(ctx, key, data, 5*time.Minute)
    
    return isps, nil
}
6. API Gateway & Load Balancing
6.1. Load Balancer Configuration (HAProxy)
haproxy
# haproxy.cfg
global
    maxconn 20000
    tune.ssl.default-dh-param 2048

frontend http-in
    bind *:80
    bind *:443 ssl crt /etc/ssl/certs/
    
    # Rate limiting per IP
    stick-table type ip size 1m expire 30s store gpc0
    http-request track-sc0 src
    http-request deny deny_status 429 if { sc0_inc_gpc0 gt 100 }
    
    default_backend api_servers

backend api_servers
    balance roundrobin
    option httpchk GET /health
    server api1 10.0.1.10:8000 check inter 3s rise 2 fall 3
    server api2 10.0.1.11:8000 check inter 3s rise 2 fall 3
    server api3 10.0.1.12:8000 check inter 3s rise 2 fall 3
    # Total 10 API servers
6.2. API Gateway (Kong)
yaml
# kong.yaml
services:
  - name: auth-service
    url: http://auth-service:8001
    routes:
      - name: auth-route
        paths:
          - /api/v1/auth
    plugins:
      - name: rate-limiting
        config:
          minute: 60
          hour: 500
      - name: cors

  - name: geospatial-service
    url: http://geospatial-service:8003
    routes:
      - name: geo-route
        paths:
          - /api/v1/maps
          - /api/v1/nearby
    plugins:
      - name: rate-limiting
        config:
          minute: 300
          hour: 1000

  - name: matching-service
    url: http://matching-service:8004
    routes:
      - name: match-route
        paths:
          - /api/v1/requests
          - /api/v1/matching
7. Message Queue & Async Processing
7.1. RabbitMQ Architecture
text
┌─────────────────────────────────────────────────────────────┐
│                    RabbitMQ Cluster (3 Nodes)              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │   Exchange  │  │   Exchange  │  │   Exchange  │       │
│  │    Job      │  │  Matching   │  │ Notification│       │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘       │
│         │                │                │               │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐       │
│  │   Queue     │  │   Queue     │  │   Queue     │       │
│  │  Job.Tech   │  │ Match.Queue │  │ Notify.Queue│       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
7.2. Key Queues
Queue Name	Purpose	Consumers	DLQ
job.assignment	Technician assignment	Matching Service (6)	Yes
job.verification	GPS/photo verification	Core Service (4)	Yes
notification.push	Push notifications	Notification Service (3)	Yes
notification.sms	SMS delivery	Notification Service (2)	Yes
analytics.job	Job analytics	Analytics Service (2)	Yes
report.generation	PDF report generation	Report Service (2)	Yes
8. Search Engine (Elasticsearch)
8.1. Elasticsearch Cluster
yaml
elasticsearch:
  cluster: production
  nodes: 5
  memory: 32GB
  disk: 2TB
  
  indices:
    - name: locations
      shards: 10
      replicas: 2
    - name: technicians
      shards: 8
      replicas: 2
    - name: isps
      shards: 6
      replicas: 2
8.2. Mapping for Fuzzy Location Search
json
{
  "mappings": {
    "properties": {
      "name": {
        "type": "text",
        "analyzer": "standard",
        "fields": {
          "fuzzy": {
            "type": "text",
            "analyzer": "standard",
            "fuzzy_transpositions": true
          },
          "phonetic": {
            "type": "text",
            "analyzer": "phonetic_analyzer"
          }
        }
      },
      "coord": {
        "type": "geo_point"
      },
      "popularity_score": {
        "type": "float"
      },
      "coverage_area": {
        "type": "geo_shape"
      }
    }
  }
}
8.3. Search Query Example
go
func (s *SearchService) SearchTechnicians(ctx context.Context, query string, lat, lng float64) ([]Technician, error) {
    esQuery := map[string]interface{}{
        "query": map[string]interface{}{
            "bool": map[string]interface{}{
                "must": []interface{}{
                    map[string]interface{}{
                        "multi_match": map[string]interface{}{
                            "query":  query,
                            "fields": []string{"name^3", "skills^2", "bio"},
                            "fuzziness": "AUTO",
                        },
                    },
                },
                "filter": []interface{}{
                    map[string]interface{}{
                        "geo_distance": map[string]interface{}{
                            "distance": "50km",
                            "location": map[string]float64{"lat": lat, "lon": lng},
                        },
                    },
                },
            },
        },
        "sort": []interface{}{
            map[string]interface{}{
                "_geo_distance": map[string]interface{}{
                    "location": map[string]float64{"lat": lat, "lon": lng},
                    "order": "asc",
                    "unit": "km",
                },
            },
        },
        "size": 50,
    }
    
    // Execute search
    result, err := s.es.Search().
        Index("technicians").
        BodyJson(esQuery).
        Do(ctx)
    // ...
}
9. WebSocket & Real-time Communication
9.1. WebSocket Architecture
text
┌──────────────────────────────────────────────────────────────┐
│                    WebSocket Gateway (4 Replicas)           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │   Connection    │  │   Connection    │                   │
│  │   Manager       │  │   Manager       │                   │
│  │   (Hash-based)  │  │   (Hash-based)  │                   │
│  └────────┬────────┘  └────────┬────────┘                   │
│           │                    │                            │
│  ┌────────▼────────────────────▼────────┐                   │
│  │         Redis Pub/Sub                 │                   │
│  │    (Cross-node communication)         │                   │
│  └───────────────────────────────────────┘                   │
└──────────────────────────────────────────────────────────────┘
9.2. Connection Manager
go
package websocket

import (
    "github.com/gorilla/websocket"
    "sync"
)

type ConnectionManager struct {
    connections map[string]*websocket.Conn
    mu          sync.RWMutex
    redis       *redis.Client
}

func (m *ConnectionManager) SendToUser(userID string, message []byte) error {
    // Check local connection
    m.mu.RLock()
    conn, exists := m.connections[userID]
    m.mu.RUnlock()
    
    if exists {
        return conn.WriteMessage(websocket.TextMessage, message)
    }
    
    // If not local, publish to Redis for other nodes
    return m.redis.Publish(context.Background(), 
        fmt.Sprintf("ws.user.%s", userID), 
        message).Err()
}
10. File Storage & Image Processing
10.1. S3 Storage Architecture
yaml
storage:
  provider: S3 (AWS/MinIO)
  buckets:
    - name: ispilo-photos
      region: us-east-1
      lifecycle:
        - rule: temp
          prefix: temp/
          days: 7
        - rule: permanent
          prefix: verified/
          days: 3650
    
    - name: ispilo-quotes
      region: us-east-1
      lifecycle:
        - rule: quotes
          days: 365
    
    - name: ispilo-portfolio
      region: us-east-1
      lifecycle:
        - rule: portfolio
          days: 3650
10.2. Image Watermarking Service
go
package image

import (
    "image"
    "image/draw"
    "image/jpeg"
    "github.com/disintegration/imaging"
)

type WatermarkService struct {
    s3Client *s3.Client
}

func (w *WatermarkService) ProcessAndWatermark(original io.Reader, metadata WatermarkMetadata) (string, error) {
    // 1. Decode image
    img, err := jpeg.Decode(original)
    if err != nil {
        return "", err
    }
    
    // 2. Resize for optimization
    resized := imaging.Resize(img, 1200, 0, imaging.Lanczos)
    
    // 3. Create watermark overlay
    overlay := w.createOverlay(metadata)
    
    // 4. Merge
    result := image.NewRGBA(resized.Bounds())
    draw.Draw(result, resized.Bounds(), resized, image.Point{}, draw.Src)
    draw.Draw(result, overlay.Bounds(), overlay, image.Point{}, draw.Over)
    
    // 5. Upload to S3
    path := fmt.Sprintf("verified/%s/%s.jpg", 
        metadata.JobID, 
        time.Now().Format("20060102_150405"))
    
    return w.uploadToS3(result, path)
}
11. Monitoring & Observability
11.1. Metrics Stack
yaml
monitoring:
  prometheus:
    scrape_interval: 15s
    retention: 30d
    
  grafana:
    dashboards:
      - system_health
      - api_performance
      - geo_queries
      - business_metrics
    
  logging:
    driver: elasticsearch
    index: ispilo-logs-{date}
    retention: 90d
    
  tracing:
    provider: Jaeger
    sampling_rate: 0.1
11.2. Key Metrics to Track
Metric	Target	Alert Threshold
API Response Time (p95)	< 200ms	> 500ms
API Error Rate	< 0.5%	> 2%
Database Connection Pool	< 80% usage	> 90%
Geospatial Query Time	< 100ms	> 300ms
Redis Cache Hit Ratio	> 80%	< 60%
WebSocket Connections	10,000	8,000 (warning)
RabbitMQ Queue Depth	< 1000	> 5000
CPU Usage (per pod)	< 70%	> 85%
12. Deployment & Scaling Strategy
12.1. Kubernetes Deployment
yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ispilo-api
spec:
  replicas: 10
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
  template:
    spec:
      containers:
      - name: api
        image: ispilo/api:latest
        ports:
        - containerPort: 8000
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        env:
        - name: DB_WRITE_URL
          valueFrom:
            secretKeyRef:
              name: db-secrets
              key: write-url
        - name: DB_READ_URL
          valueFrom:
            secretKeyRef:
              name: db-secrets
              key: read-url
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ispilo-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ispilo-api
  minReplicas: 5
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    metric:
      name: requests_per_second
    target:
      type: AverageValue
      averageValue: "1000"
12.2. Vertical Scaling (DB)
yaml
db_resources:
  primary:
    cpu: 16
    memory: 64Gi
    storage: 1TB (NVMe)
  
  replicas:
    - name: geo-replica
      cpu: 16
      memory: 64Gi
      storage: 1TB
    - name: reporting-replica
      cpu: 8
      memory: 32Gi
      storage: 2TB
13. Disaster Recovery & High Availability
13.1. Backup Strategy
yaml
backup:
  postgres:
    full_backup: daily at 2 AM
    wal_archive: every 15 minutes
    retention: 30 days
    location: s3://backups/postgres/
  
  redis:
    snapshot: every 1 hour
    retention: 7 days
    location: s3://backups/redis/
  
  elasticsearch:
    snapshot: daily
    retention: 30 days
    location: s3://backups/elasticsearch/
13.2. Failover Scenarios
Scenario	Action	RTO	RPO
Primary DB Failure	Promote Replica 1 to Primary	5 min	5 min
Region Failure	Failover to DR region	30 min	15 min
API Pod Crash	K8s auto-restart	30 sec	0
Cache Failure	Fallback to DB	1 sec	0
Message Queue Failure	Drain to secondary cluster	2 min	1 min
14. Security Implementation
14.1. Authentication Flow
text
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Mobile  │     │  Auth    │     │  Redis   │     │  Postgres│
│  Client  │────▶│  Service │────▶│  Store   │────▶│  Verify  │
│          │     │          │     │  Token   │     │  User    │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
     │                │                │                │
     │  1. Phone+OTP  │                │                │
     │───────────────▶│                │                │
     │                │  2. Verify OTP │                │
     │                │───────────────▶│                │
     │                │                │  3. Get User   │
     │                │───────────────────────────────▶│
     │                │                │                │
     │  4. JWT Token  │                │                │
     │◀───────────────│                │                │
     │                │  5. Store JWT  │                │
     │                │───────────────▶│                │
     │                │                │                │
14.2. Security Hardening
yaml
security:
  jwt:
    algorithm: RS256
    access_token_ttl: 15m
    refresh_token_ttl: 30d
    blacklist_redis: true
  
  encryption:
    at_rest: AES-256 (S3 server-side)
    in_transit: TLS 1.3
    database: Column-level (pgcrypto)
  
  rate_limiting:
    auth_endpoints: 5/minute
    api_endpoints: 300/minute
    search_endpoints: 100/minute
  
  fraud_detection:
    gps_anomaly_threshold: 100km
    multiple_accounts: Same device ID detection
    photo_reuse: Perceptual hashing
15. Cost Estimation (Mo