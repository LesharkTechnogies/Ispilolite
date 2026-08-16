package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"log"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v2"
)

// NewPostgresConnection creates a new PostgreSQL connection using sqlx.
func NewPostgresConnection(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// DBConfig holds the configuration for a single database connection.
type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

// Config holds the full database configuration for reader and writer.
type Config struct {
	Database struct {
		PostgresWriter DBConfig `yaml:"postgres_writer"`
		PostgresReader DBConfig `yaml:"postgres_reader"`
	} `yaml:"database"`
}

var (
	// DBWriter is the connection to the primary write database.
	DBWriter *sql.DB
	// DBReader is the connection to the read-replica database.
	DBReader *sql.DB
)

// InitDB initializes the database connections.
func InitDB(configPath string) error {
	loadDotEnv(".env")
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	configFile = []byte(os.ExpandEnv(string(configFile)))
	var config Config
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	writerDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		config.Database.PostgresWriter.Host,
		config.Database.PostgresWriter.Port,
		config.Database.PostgresWriter.User,
		config.Database.PostgresWriter.Password,
		config.Database.PostgresWriter.DBName,
	)

	readerDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		config.Database.PostgresReader.Host,
		config.Database.PostgresReader.Port,
		config.Database.PostgresReader.User,
		config.Database.PostgresReader.Password,
		config.Database.PostgresReader.DBName,
	)
	// Neon provides a complete pooled connection string via DATABASE_URL.
	// Use it for both pools unless an explicit reader URL is supplied.
	if url := os.Getenv("DATABASE_URL"); url != "" {
		writerDSN = url
	}
	if url := os.Getenv("DB_URL"); url != "" {
		writerDSN = url
	}
	if url := os.Getenv("DATABASE_READ_URL"); url != "" {
		readerDSN = url
	} else if url := os.Getenv("DB_READ_URL"); url != "" {
		readerDSN = url
	} else if os.Getenv("DATABASE_URL") != "" || os.Getenv("DB_URL") != "" {
		readerDSN = writerDSN
	}

	DBWriter, err = sql.Open("postgres", writerDSN)
	if err != nil {
		return fmt.Errorf("failed to open writer database: %w", err)
	}
	fmt.Println("Successfully connected to the writer database.")

	DBReader, err = sql.Open("postgres", readerDSN)
	if err != nil {
		if DBWriter != nil { _ = DBWriter.Close() }
		return fmt.Errorf("failed to open reader database: %w", err)
	}

	// Bound the pools so a burst of traffic across many API replicas cannot
	// exhaust Postgres connection slots. Writer traffic is lighter than read
	// traffic, so the reader pool is sized larger. Tune via env for the target
	// deployment (max_connections / replica count).
	configurePool(DBWriter, "DB_WRITER_MAX_CONNS", 25)
	configurePool(DBReader, "DB_READER_MAX_CONNS", 50)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := DBWriter.PingContext(ctx); err != nil {
		_ = DBWriter.Close(); _ = DBReader.Close()
		return fmt.Errorf("failed to ping writer database: %w", err)
	}
	if err := DBReader.PingContext(ctx); err != nil {
		_ = DBWriter.Close(); _ = DBReader.Close()
		return fmt.Errorf("failed to ping reader database: %w", err)
	}
	if err := runMigrations(ctx, getenv("MIGRATIONS_PATH", "migrations")); err != nil {
		_ = DBWriter.Close(); _ = DBReader.Close()
		return fmt.Errorf("database migrations failed: %w", err)
	}
	if err := verifySchema(ctx, DBWriter); err != nil {
		_ = DBWriter.Close()
		_ = DBReader.Close()
		return fmt.Errorf("writer schema verification failed: %w", err)
	}
	if err := verifySchema(ctx, DBReader); err != nil {
		_ = DBWriter.Close()
		_ = DBReader.Close()
		return fmt.Errorf("reader schema verification failed: %w", err)
	}
	log.Println("database connection successful; schema ready")
	return nil
}

func runMigrations(ctx context.Context, dir string) error {
	conn, err := DBWriter.Conn(ctx)
	if err != nil { return fmt.Errorf("acquire migration connection: %w", err) }
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock(821527390)"); err != nil { return fmt.Errorf("acquire migration lock: %w", err) }
	defer conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock(821527390)")
	if _,err:=conn.ExecContext(ctx,`CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY,checksum TEXT NOT NULL,applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`);err!=nil{return fmt.Errorf("create schema_migrations: %w",err)}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql")); if err != nil { return err }
	sort.Strings(files)
	for _, file := range files { sqlBytes, err := os.ReadFile(file); if err != nil { return err }; migrationSQL:=migrationBody(string(sqlBytes));if migrationSQL==""{continue};version:=filepath.Base(file);sum:=sha256.Sum256(sqlBytes);checksum:=hex.EncodeToString(sum[:]);var existing string;err=conn.QueryRowContext(ctx,`SELECT checksum FROM schema_migrations WHERE version=$1`,version).Scan(&existing);if err==nil{if existing!=checksum{return fmt.Errorf("migration %s changed after being applied",version)};continue};if err!=sql.ErrNoRows{return err};tx,err:=conn.BeginTx(ctx,nil);if err!=nil{return err};if _,err=tx.ExecContext(ctx,migrationSQL);err!=nil{tx.Rollback();return fmt.Errorf("%s: %w",file,err)};if _,err=tx.ExecContext(ctx,`INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)`,version,checksum);err!=nil{tx.Rollback();return err};if err=tx.Commit();err!=nil{return err} }
	if len(files) == 0 { return fmt.Errorf("no migration files found in %s", dir) }
	return nil
}

func migrationBody(raw string) string {
	lines:=strings.Split(strings.ReplaceAll(raw,"\r\n","\n"),"\n")
	start,end:=0,len(lines)
	for start<end&&strings.TrimSpace(lines[start])==""{start++}
	if start<end&&strings.EqualFold(strings.TrimSpace(lines[start]),"BEGIN;"){start++}
	for end>start&&strings.TrimSpace(lines[end-1])==""{end--}
	if end>start&&strings.EqualFold(strings.TrimSpace(lines[end-1]),"COMMIT;"){end--}
	return strings.TrimSpace(strings.Join(lines[start:end],"\n"))
}

func verifySchema(ctx context.Context, db *sql.DB) error {
	const q = `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('users','isps','installations','locations')`
	var count int; if err := db.QueryRowContext(ctx, q).Scan(&count); err != nil { return err }; if count < 4 { return fmt.Errorf("expected core tables, found %d/4", count) }; return nil
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil { return }
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") { continue }
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 { continue }
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if key != "" && os.Getenv(key) == "" { _ = os.Setenv(key, value) }
	}
}

func getenv(key, fallback string) string { if value := os.Getenv(key); value != "" { return value }; return fallback }

// GetWriter returns the database connection for write operations.
func GetWriter() *sql.DB {
	return DBWriter
}

// GetReader returns the database connection for read operations.
func GetReader() *sql.DB {
	return DBReader
}

// CloseDB releases all PostgreSQL pools during graceful shutdown.
func CloseDB() error {
	var firstErr error
	if DBReader != nil && DBReader != DBWriter {
		if err := DBReader.Close(); err != nil { firstErr = err }
	}
	if DBWriter != nil {
		if err := DBWriter.Close(); err != nil && firstErr == nil { firstErr = err }
	}
	return firstErr
}

// configurePool applies connection-pool limits to a database handle. The maximum
// open connections can be overridden per-pool via an environment variable so the
// same binary can be tuned for different replica counts without a rebuild.
func configurePool(db *sql.DB, maxConnsEnv string, defaultMax int) {
	maxOpen := defaultMax
	if value := os.Getenv(maxConnsEnv); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			maxOpen = parsed
		}
	}

	db.SetMaxOpenConns(maxOpen)
	// Keep a warm pool of idle connections but never more than a quarter of the
	// open limit, so idle replicas release connections back to Postgres.
	idle := maxOpen / 4
	if idle < 2 {
		idle = 2
	}
	db.SetMaxIdleConns(idle)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
}
