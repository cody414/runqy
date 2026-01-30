package monitoring

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
)

// DatabaseInfo represents the database connection information.
type DatabaseInfo struct {
	Type      string         `json:"type"`
	Connected bool           `json:"connected"`
	Host      string         `json:"host,omitempty"`
	Database  string         `json:"database,omitempty"`
	Stats     *DatabaseStats `json:"stats,omitempty"`
}

// DatabaseStats contains database statistics.
type DatabaseStats struct {
	OpenConnections int `json:"open_connections"`
	InUse           int `json:"in_use"`
	Idle            int `json:"idle"`
}

func newDatabaseInfoHandlerFunc(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		info := DatabaseInfo{
			Connected: false,
		}

		if db == nil {
			json.NewEncoder(w).Encode(info)
			return
		}

		// Determine database type from driver name
		driverName := db.DriverName()
		switch driverName {
		case "sqlite3", "sqlite":
			info.Type = "SQLite"
			// For SQLite, get the database file path from environment or default
			dbPath := os.Getenv("SQLITE_DB_PATH")
			if dbPath == "" {
				dbPath = "runqy.db"
			}
			info.Database = filepath.Base(dbPath)
		case "postgres", "pgx":
			info.Type = "PostgreSQL"
			info.Host = os.Getenv("DATABASE_HOST")
			if info.Host == "" {
				info.Host = "localhost"
			}
			port := os.Getenv("DATABASE_PORT")
			if port != "" && port != "5432" {
				info.Host = info.Host + ":" + port
			}
			info.Database = os.Getenv("DATABASE_DBNAME")
			if info.Database == "" {
				info.Database = "runqy"
			}
		default:
			info.Type = driverName
		}

		// Check connection and get stats
		if err := db.Ping(); err == nil {
			info.Connected = true

			// Get connection pool stats
			stats := db.Stats()
			info.Stats = &DatabaseStats{
				OpenConnections: stats.OpenConnections,
				InUse:           stats.InUse,
				Idle:            stats.Idle,
			}
		}

		if err := json.NewEncoder(w).Encode(info); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
