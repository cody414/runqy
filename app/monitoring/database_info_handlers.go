package monitoring

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/Publikey/runqy/config"
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

func newDatabaseInfoHandlerFunc(db *sqlx.DB, cfg *config.Config) http.HandlerFunc {
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
			dbPath := "runqy.db"
			if cfg != nil && cfg.SQLiteDBPath != "" {
				dbPath = cfg.SQLiteDBPath
			}
			info.Database = filepath.Base(dbPath)
		case "postgres", "pgx":
			info.Type = "PostgreSQL"
			info.Host = "localhost"
			if cfg != nil && cfg.PostgresHost != "" {
				info.Host = cfg.PostgresHost
			}
			if cfg != nil && cfg.PostgresPort != "" && cfg.PostgresPort != "5432" {
				info.Host = info.Host + ":" + cfg.PostgresPort
			}
			info.Database = "runqy"
			if cfg != nil && cfg.PostgresDB != "" {
				info.Database = cfg.PostgresDB
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
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
