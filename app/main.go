package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Publikey/runqy/api"
	"github.com/Publikey/runqy/config"
	_ "github.com/Publikey/runqy/docs"
	"github.com/Publikey/runqy/models"
	queueworker "github.com/Publikey/runqy/queues"
	"github.com/Publikey/runqy/watcher"

	"github.com/Publikey/runqy/monitoring"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//	@title			Runqy Queue API
//	@version		1.0.0
//	@description	Task queueing service built on asynq for managing distributed task processing

var wg sync.WaitGroup

func main() {
	// Parse CLI flags
	configDir := flag.String("config", "", "Path to queue workers config directory (overrides QUEUE_WORKERS_DIR)")
	enableWatch := flag.Bool("watch", false, "Enable file/git watching for config auto-reload")
	configRepo := flag.String("config-repo", "", "GitHub repo URL for configs (e.g., https://github.com/org/repo.git)")
	configBranch := flag.String("config-branch", "", "Git branch (default: main)")
	configPath := flag.String("config-path", "", "Path within repo to YAML files")
	cloneDir := flag.String("clone-dir", "", "Directory to clone repo into (default: downloads)")
	watchInterval := flag.Int("watch-interval", 0, "Git polling interval in seconds (default: 60)")
	flag.Parse()

	// Load .env file if present (for local development)
	// Looks for .env.secret in parent dir, then .env in current dir
	if err := godotenv.Load("../.env.secret"); err != nil {
		_ = godotenv.Load() // Try .env as fallback
	}

	// Load configuration
	cfg := config.Load()

	// Override config from CLI flags
	if *configDir != "" {
		cfg.QueueWorkersDir = *configDir
	}
	if *configRepo != "" {
		cfg.ConfigRepoURL = *configRepo
	}
	if *configBranch != "" {
		cfg.ConfigRepoBranch = *configBranch
	}
	if *configPath != "" {
		cfg.ConfigRepoPath = *configPath
	}
	if *cloneDir != "" {
		cfg.ConfigCloneDir = *cloneDir
	}
	if *watchInterval > 0 {
		cfg.ConfigWatchInterval = *watchInterval
	}

	// Create API router
	router := gin.Default()
	router.Use(cors.Default())

	redisAddr, err := models.BuildRedisConns()
	if err != nil {
		log.Fatalf("[FATAL] Redis build connection failed: %v", err)
	}

	// Build PostgreSQL connection for queue worker configs
	pgDB, err := models.BuildPostgresDB(cfg)
	if err != nil {
		log.Fatalf("[FATAL] PostgreSQL connection failed: %v", err)
	}
	defer pgDB.Close()
	log.Println("[INFO] PostgreSQL connection established")

	// Ensure database schema exists (creates tables if missing)
	if err := models.EnsureSchema(pgDB); err != nil {
		log.Fatalf("[FATAL] Failed to initialize database schema: %v", err)
	}

	redisClient := asynq.NewClient(redisAddr.AsynqOpt)
	defer redisClient.Close()

	router.Use(func(c *gin.Context) {
		c.Set("client", redisClient)
		c.Set("rdb", redisAddr.RDB)
		c.Next()
	})

	h := monitoring.New(monitoring.Options{
		RootPath:     "/monitoring",
		RedisConnOpt: redisAddr.AsynqOpt,
		ReadOnly:     os.Getenv("ASYNQ_READ_ONLY") == "true",
	})

	// Initialize queue worker store (PostgreSQL for configs, Redis for asynq)
	qwStore := queueworker.NewStore(pgDB, redisAddr.RDB)

	// Clean up stale workers from previous runs
	ctx := context.Background()
	if deleted, err := qwStore.CleanupStaleWorkers(ctx); err != nil {
		log.Printf("[WARN] Failed to cleanup stale workers: %v", err)
	} else if deleted > 0 {
		log.Printf("[CLEANUP] Removed %d stale worker entries", deleted)
	}

	// Initialize git loader if repo URL is configured
	var gitLoader *queueworker.GitLoader
	if cfg.ConfigRepoURL != "" {
		var gitErr error
		gitLoader, gitErr = queueworker.NewGitLoader(cfg)
		if gitErr != nil {
			log.Fatalf("[FATAL] Failed to initialize git loader: %v", gitErr)
		}
		if cloneErr := gitLoader.Clone(); cloneErr != nil {
			log.Fatalf("[FATAL] Failed to clone config repo: %v", cloneErr)
		}
		defer gitLoader.Cleanup()

		// Use git repo path instead of local dir
		cfg.QueueWorkersDir = gitLoader.GetConfigPath()
		log.Printf("[GIT-LOADER] Using configs from: %s", cfg.QueueWorkersDir)
	}

	if _, err := api.LoadQueueWorkersAtStartup(qwStore, cfg.QueueWorkersDir); err != nil {
		log.Printf("[WARN] Failed to load queue workers: %v", err)
	}

	// Start config watcher if enabled
	var configWatcher *watcher.ConfigWatcher
	var gitWatcher *watcher.GitWatcher

	if *enableWatch {
		reloadFunc := func(ctx context.Context) ([]string, []string) {
			return api.ReloadFromYAMLContext(ctx, qwStore, cfg.QueueWorkersDir)
		}

		if gitLoader != nil {
			// Use git polling watcher
			interval := time.Duration(cfg.ConfigWatchInterval) * time.Second
			gitWatcher = watcher.NewGitWatcher(gitLoader.Pull, reloadFunc, interval)
			if err := gitWatcher.Start(); err != nil {
				log.Printf("[WARN] Failed to start git watcher: %v", err)
				gitWatcher = nil
			}
		} else {
			// Use filesystem watcher
			var watchErr error
			configWatcher, watchErr = watcher.NewConfigWatcher(cfg.QueueWorkersDir, reloadFunc)
			if watchErr != nil {
				log.Printf("[WARN] Failed to create config watcher: %v", watchErr)
			} else {
				if startErr := configWatcher.Start(); startErr != nil {
					log.Printf("[WARN] Failed to start config watcher: %v", startErr)
					configWatcher = nil
				}
			}
		}
	}

	api.SetupAPI(router, qwStore, cfg.QueueWorkersDir, cfg)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.StaticFile("/swagger.yaml", "./docs/swagger.yaml")

	router.Any("/monitoring/*a", gin.WrapH(h))

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: router,
	}

	// Handle OS interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// Start the HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("[INFO] HTTP server starting on :%s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-quit
	log.Println("[INFO] Shutdown signal received")

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
	log.Println("[INFO] All goroutines exited")
}
