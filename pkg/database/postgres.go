package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"

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

// InitDB initializes the database connections for both writer and reader.
func InitDB(configPath string) {
	// Read and parse the configuration file
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load database configuration: %v", err)
	}

	// Initialize writer connection
	writerDSN := buildDSN(config.Database.PostgresWriter)
	DBWriter, err = sql.Open("postgres", writerDSN)
	if err != nil {
		log.Fatalf("Failed to open writer database connection: %v", err)
	}
	if err := DBWriter.Ping(); err != nil {
		log.Fatalf("Failed to ping writer database: %v", err)
	}
	fmt.Println("Successfully connected to the writer database.")

	// Initialize reader connection
	readerDSN := buildDSN(config.Database.PostgresReader)
	DBReader, err = sql.Open("postgres", readerDSN)
	if err != nil {
		log.Fatalf("Failed to open reader database connection: %v", err)

	}
	if err := DBReader.Ping(); err != nil {
		log.Fatalf("Failed to ping reader database: %v", err)
	}
	fmt.Println("Successfully connected to the reader database.")
}

// loadConfig reads the yaml file from the given path and unmarshals it into a Config struct.
func loadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// buildDSN constructs the data source name string for a postgres connection.
func buildDSN(dbConfig DBConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName)
}

// GetWriter returns the database connection for write operations.
func GetWriter() *sql.DB {
	return DBWriter
}

// GetReader returns the database connection for read operations.
func GetReader() *sql.DB {
	return DBReader
}
