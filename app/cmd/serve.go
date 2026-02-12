package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Publikey/runqy/api"
	"github.com/Publikey/runqy/auth"
	_ "github.com/Publikey/runqy/docs"
	"github.com/Publikey/runqy/models"
	"github.com/Publikey/runqy/monitoring"
	queueworker "github.com/Publikey/runqy/queues"
	"github.com/Publikey/runqy/vaults"
	"github.com/Publikey/runqy/watcher"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/hibiken/asynq/x/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	configDir     string
	enableWatch   bool
	configRepo    string
	configBranch  string
	configPath    string
	cloneDir      string
	watchInterval int
	useSQLite     bool
	disableUI     bool
	debugMode     bool
)

// DebugMode is a package-level variable that can be checked by other packages
var DebugMode = false

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the runqy HTTP server",
	Long: `Start the runqy HTTP server for worker registration, queue management,
REST API, and monitoring dashboard.

Example:
  runqy serve
  runqy serve --config ./deployment --watch
  runqy serve --config-repo https://github.com/org/repo.git --watch
  runqy serve --sqlite    # Use SQLite instead of PostgreSQL (for testing)`,
	Run: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Serve-specific flags
	serveCmd.Flags().StringVar(&configDir, "config", "", "Path to queue workers config directory (overrides QUEUE_WORKERS_DIR)")
	serveCmd.Flags().BoolVar(&enableWatch, "watch", false, "Enable file/git watching for config auto-reload")
	serveCmd.Flags().StringVar(&configRepo, "config-repo", "", "GitHub repo URL for configs (e.g., https://github.com/org/repo.git)")
	serveCmd.Flags().StringVar(&configBranch, "config-branch", "", "Git branch (default: main)")
	serveCmd.Flags().StringVar(&configPath, "config-path", "", "Path within repo to YAML files")
	serveCmd.Flags().StringVar(&cloneDir, "clone-dir", "", "Directory to clone repo into (default: downloads)")
	serveCmd.Flags().IntVar(&watchInterval, "watch-interval", 0, "Git polling interval in seconds (default: 60)")
	serveCmd.Flags().BoolVar(&useSQLite, "sqlite", false, "Use SQLite instead of PostgreSQL (for testing, NOT recommended for production)")
	serveCmd.Flags().BoolVar(&disableUI, "no-ui", false, "Disable the monitoring web dashboard")
	serveCmd.Flags().BoolVar(&debugMode, "debug", false, "Enable verbose logging (GIN routes, detailed startup logs)")
}

func runServe(cmd *cobra.Command, args []string) {
	var wg sync.WaitGroup

	// Set package-level debug mode for other packages to check
	DebugMode = debugMode

	cfg := GetConfig()

	// Override config from CLI flags
	if configDir != "" {
		cfg.QueueWorkersDir = configDir
	}
	if configRepo != "" {
		cfg.ConfigRepoURL = configRepo
	}
	if configBranch != "" {
		cfg.ConfigRepoBranch = configBranch
	}
	if configPath != "" {
		cfg.ConfigRepoPath = configPath
	}
	if cloneDir != "" {
		cfg.ConfigCloneDir = cloneDir
	}
	if watchInterval > 0 {
		cfg.ConfigWatchInterval = watchInterval
	}
	if useSQLite {
		cfg.UseSQLite = true
	}

	// Initialize startup config for banner
	startupCfg := StartupConfig{
		Version:      Version,
		Port:         cfg.HTTPPort,
		UIEnabled:    !disableUI,
		WatchEnabled: enableWatch,
		GitRepoURL:   cfg.ConfigRepoURL,
	}

	// Create API router - use release mode by default to suppress GIN logs
	var router *gin.Engine
	if debugMode {
		gin.SetMode(gin.DebugMode)
		router = gin.Default() // Includes logger and recovery
	} else {
		gin.SetMode(gin.ReleaseMode)
		router = gin.New()
		router.Use(gin.Recovery()) // Keep recovery but no logger
	}
	router.Use(cors.Default())

	redisAddr, err := models.BuildRedisConns()
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	startupCfg.RedisHost = cfg.RedisHost + ":" + cfg.RedisPort
	startupCfg.RedisConnected = true

	// Build database connection for queue worker configs (PostgreSQL or SQLite)
	db, err := models.BuildDB(cfg)
	if err != nil {
		if cfg.UseSQLite {
			log.Fatalf("SQLite connection failed: %v", err)
		} else {
			log.Fatalf("PostgreSQL connection failed: %v", err)
		}
	}
	defer db.Close()

	if cfg.UseSQLite {
		startupCfg.DatabaseType = "SQLite"
		startupCfg.DatabaseName = cfg.SQLiteDBPath
		if debugMode {
			log.Println("WARNING: Using SQLite database. This is NOT recommended for production!")
		}
	} else {
		startupCfg.DatabaseType = "PostgreSQL"
		startupCfg.DatabaseName = cfg.PostgresDB
		if debugMode {
			log.Println("[INFO] PostgreSQL connection established")
		}
	}

	// Ensure database schema exists (creates tables if missing)
	if err := models.EnsureSchema(db, debugMode); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	redisClient := asynq.NewClient(redisAddr.AsynqOpt)
	defer redisClient.Close()

	router.Use(func(c *gin.Context) {
		c.Set("client", redisClient)
		c.Set("rdb", redisAddr.RDB)
		c.Next()
	})

	// Initialize vault store
	vaultStore := vaults.NewStore(db)
	startupCfg.VaultsEnabled = vaultStore.IsEnabled()
	if debugMode {
		if vaultStore.IsEnabled() {
			log.Println("[VAULTS] Vaults feature enabled")
		} else {
			log.Println("[VAULTS] Warning: RUNQY_VAULT_MASTER_KEY not set, vaults feature disabled")
		}
	}

	// Register asynq metrics exporter for Prometheus
	inspector := asynq.NewInspector(redisAddr.AsynqOpt)
	defer inspector.Close()
	metricsExporter := metrics.NewQueueMetricsCollector(inspector)
	prometheus.MustRegister(metricsExporter)
	if debugMode {
		log.Println("[METRICS] Prometheus metrics exporter registered at /metrics")
	}

	// Initialize queue worker store (database for configs, Redis for asynq)
	qwStore := queueworker.NewStore(db, redisAddr.RDB)

	// Initialize monitoring UI (unless disabled)
	var h *monitoring.HTTPHandler
	if !disableUI {
		authStore := auth.NewStore(db)
		if debugMode {
			log.Println("[AUTH] Monitoring UI authentication enabled")
		}

		h = monitoring.New(monitoring.Options{
			RootPath:          "/monitoring",
			RedisConnOpt:      redisAddr.AsynqOpt,
			ReadOnly:          os.Getenv("ASYNQ_READ_ONLY") == "true",
			DB:                db,
			VaultStore:        vaultStore,
			QueueStore:        qwStore,
			AuthStore:         authStore,
			PrometheusAddress: os.Getenv("PROMETHEUS_ADDRESS"),
		})
		defer h.Close()
	} else if debugMode {
		log.Println("[UI] Monitoring dashboard disabled (--no-ui)")
	}

	// Clean up stale workers from previous runs
	ctx := context.Background()
	if deleted, err := qwStore.CleanupStaleWorkers(ctx); err != nil {
		startupCfg.RedisConnected = false
		if debugMode {
			log.Printf("[WARN] Redis connection failed (%s:%s): unable to cleanup stale workers", cfg.RedisHost, cfg.RedisPort)
		}
	} else if deleted > 0 && debugMode {
		log.Printf("[CLEANUP] Removed %d stale worker entries", deleted)
	}

	// Initialize git loader if repo URL is configured
	var gitLoader *queueworker.GitLoader
	if cfg.ConfigRepoURL != "" {
		var gitErr error
		gitLoader, gitErr = queueworker.NewGitLoader(cfg)
		if gitErr != nil {
			log.Fatalf("Failed to initialize git loader: %v", gitErr)
		}
		if cloneErr := gitLoader.Clone(); cloneErr != nil {
			log.Fatalf("Failed to clone config repo: %v", cloneErr)
		}
		defer gitLoader.Cleanup()

		// Use git repo path instead of local dir
		cfg.QueueWorkersDir = gitLoader.GetConfigPath()
		if debugMode {
			log.Printf("[GIT-LOADER] Using configs from: %s", cfg.QueueWorkersDir)
		}
	}

	queuesLoaded, err := api.LoadQueueWorkersAtStartup(qwStore, cfg.QueueWorkersDir, debugMode)
	if err != nil && debugMode {
		log.Printf("[WARN] Failed to load queue workers: %v", err)
	}
	startupCfg.QueuesLoaded = len(queuesLoaded)

	// Start config watcher if enabled
	var configWatcher *watcher.ConfigWatcher
	var gitWatcher *watcher.GitWatcher

	if enableWatch {
		reloadFunc := func(ctx context.Context) ([]string, []string) {
			return api.ReloadFromYAMLContext(ctx, qwStore, cfg.QueueWorkersDir)
		}

		if gitLoader != nil {
			// Use git polling watcher
			interval := time.Duration(cfg.ConfigWatchInterval) * time.Second
			gitWatcher = watcher.NewGitWatcher(gitLoader.Pull, reloadFunc, interval)
			if err := gitWatcher.Start(); err != nil {
				if debugMode {
					log.Printf("[WARN] Failed to start git watcher: %v", err)
				}
				gitWatcher = nil
			}
		} else {
			// Use filesystem watcher
			var watchErr error
			configWatcher, watchErr = watcher.NewConfigWatcher(cfg.QueueWorkersDir, reloadFunc)
			if watchErr != nil {
				if debugMode {
					log.Printf("[WARN] Failed to create config watcher: %v", watchErr)
				}
			} else {
				if startErr := configWatcher.Start(); startErr != nil {
					if debugMode {
						log.Printf("[WARN] Failed to start config watcher: %v", startErr)
					}
					configWatcher = nil
				}
			}
		}
	}

	// Health check endpoints (no auth required)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	router.GET("/healthz", func(c *gin.Context) {
		// Kubernetes-style health check with dependency status
		redisOK := redisAddr.RDB.Ping(c).Err() == nil

		status := "ok"
		httpCode := 200
		if !redisOK {
			status = "degraded"
			httpCode = 503
		}

		c.JSON(httpCode, gin.H{
			"status":    status,
			"redis":     redisOK,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	api.SetupAPI(router, qwStore, cfg.QueueWorkersDir, cfg, redisAddr.AsynqOpt)
	api.SetupVaultsAPI(router, vaultStore)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.StaticFile("/swagger.yaml", "./docs/swagger.yaml")

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Redirect root to monitoring dashboard (unless disabled)
	if !disableUI {
		router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/monitoring")
		})
		router.Any("/monitoring/*a", gin.WrapH(h))
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: router,
	}

	// Handle OS shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Print startup banner
	PrintStartupBanner(startupCfg)

	// Start the HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sig := <-quit
	log.Printf("Received %v, shutting down...", sig)

	// Stop watchers
	if configWatcher != nil {
		configWatcher.Stop()
	}
	if gitWatcher != nil {
		gitWatcher.Stop()
	}

	// Graceful shutdown of HTTP server
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		log.Fatalf("HTTP server forced to shutdown: %v", err)
	}

	wg.Wait()
	log.Println("Server shutdown complete")
}
