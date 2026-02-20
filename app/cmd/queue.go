package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Publikey/runqy/models"
	queueworker "github.com/Publikey/runqy/queues"
	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
)

// queueCmd represents the queue command
var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Queue management commands",
	Long: `Queue management commands for listing, inspecting, and managing queues.

Examples:
  runqy queue list
  runqy queue inspect inference_high
  runqy queue pause inference_high
  runqy queue unpause inference_high

Remote mode:
  runqy --server https://runqy.example.com -k API_KEY queue list`,
}

// queueListCmd lists all queues with stats
var queueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all queues with statistics",
	Long: `List all queues registered in asynq with their current statistics.

Shows pending, active, scheduled, retry, archived, and completed task counts.`,
	RunE: runQueueList,
}

// queueInspectCmd shows detailed info about a queue
var queueInspectCmd = &cobra.Command{
	Use:   "inspect <queue>",
	Short: "Show detailed queue information",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueInspect,
}

// queuePauseCmd pauses a queue
var queuePauseCmd = &cobra.Command{
	Use:   "pause <queue>",
	Short: "Pause a queue (stops task processing)",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueuePause,
}

// queueUnpauseCmd unpauses a queue
var queueUnpauseCmd = &cobra.Command{
	Use:   "unpause <queue>",
	Short: "Unpause a queue (resumes task processing)",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueUnpause,
}

func init() {
	rootCmd.AddCommand(queueCmd)
	queueCmd.AddCommand(queueListCmd)
	queueCmd.AddCommand(queueInspectCmd)
	queueCmd.AddCommand(queuePauseCmd)
	queueCmd.AddCommand(queueUnpauseCmd)
}

func getInspector() (*asynq.Inspector, error) {
	redisAddr, err := models.BuildRedisConns(GetConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	return asynq.NewInspector(redisAddr.AsynqOpt), nil
}

func runQueueList(cmd *cobra.Command, args []string) error {
	// Remote mode: use API client
	if IsRemoteMode() {
		return runQueueListRemote()
	}

	// Local mode: direct Redis access
	// First, sync configs from PostgreSQL to asynq
	cfg := GetConfig()
	pgDB, err := models.BuildPostgresDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer pgDB.Close()

	redisConns, err := models.BuildRedisConns(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	defer redisConns.RDB.Close()

	store := queueworker.NewStore(pgDB, redisConns.RDB)
	if err := store.SyncConfigsToAsynq(context.Background()); err != nil {
		fmt.Printf("Warning: failed to sync configs to asynq: %v\n", err)
	}

	inspector := asynq.NewInspector(redisConns.AsynqOpt)
	defer inspector.Close()

	queues, err := inspector.Queues()
	if err != nil {
		return fmt.Errorf("failed to list queues: %w", err)
	}

	if len(queues) == 0 {
		fmt.Println("No queues found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "QUEUE\tPENDING\tACTIVE\tSCHEDULED\tRETRY\tARCHIVED\tCOMPLETED\tPAUSED")

	for _, queueName := range queues {
		info, err := inspector.GetQueueInfo(queueName)
		if err != nil {
			fmt.Fprintf(w, "%s\t-\t-\t-\t-\t-\t-\t-\n", queueName)
			continue
		}

		paused := "no"
		if info.Paused {
			paused = "yes"
		}

		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n",
			info.Queue,
			info.Pending,
			info.Active,
			info.Scheduled,
			info.Retry,
			info.Archived,
			info.Completed,
			paused,
		)
	}

	w.Flush()
	return nil
}

func runQueueListRemote() error {
	client := NewAPIClient()

	queues, err := client.ListQueues()
	if err != nil {
		return err
	}

	if len(queues) == 0 {
		fmt.Println("No queues found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "QUEUE\tPENDING\tACTIVE\tSCHEDULED\tRETRY\tARCHIVED\tCOMPLETED\tPAUSED")

	for _, info := range queues {
		paused := "no"
		if info.Paused {
			paused = "yes"
		}

		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n",
			info.Queue,
			info.Pending,
			info.Active,
			info.Scheduled,
			info.Retry,
			info.Archived,
			info.Completed,
			paused,
		)
	}

	w.Flush()
	return nil
}

func runQueueInspect(cmd *cobra.Command, args []string) error {
	if err := validateArgs(args, "queue name"); err != nil {
		return err
	}
	queueName := args[0]

	// Remote mode: use API client
	if IsRemoteMode() {
		return runQueueInspectRemote(queueName)
	}

	// Local mode: direct Redis access
	inspector, err := getInspector()
	if err != nil {
		return err
	}
	defer inspector.Close()

	info, err := inspector.GetQueueInfo(queueName)
	if err != nil {
		return fmt.Errorf("failed to get queue info: %w", err)
	}

	fmt.Printf("Queue: %s\n", info.Queue)
	fmt.Printf("Paused: %v\n", info.Paused)
	fmt.Printf("Memory Usage: %d bytes\n", info.MemoryUsage)
	fmt.Println()
	fmt.Println("Task Counts:")
	fmt.Printf("  Pending:   %d\n", info.Pending)
	fmt.Printf("  Active:    %d\n", info.Active)
	fmt.Printf("  Scheduled: %d\n", info.Scheduled)
	fmt.Printf("  Retry:     %d\n", info.Retry)
	fmt.Printf("  Archived:  %d\n", info.Archived)
	fmt.Printf("  Completed: %d\n", info.Completed)
	fmt.Printf("  Aggregating: %d\n", info.Aggregating)
	fmt.Println()
	fmt.Printf("Processed Today: %d\n", info.ProcessedTotal)
	fmt.Printf("Failed Today:    %d\n", info.FailedTotal)

	// Show daily stats
	fmt.Println()
	fmt.Println("Recent Processing Stats (last 7 days):")
	dailyStats, err := inspector.History(queueName, 7)
	if err == nil && len(dailyStats) > 0 {
		for _, stat := range dailyStats {
			fmt.Printf("  %s: processed=%d, failed=%d\n",
				stat.Date.Format("2006-01-02"),
				stat.Processed,
				stat.Failed,
			)
		}
	} else {
		fmt.Println("  No history available")
	}

	return nil
}

func runQueueInspectRemote(queueName string) error {
	client := NewAPIClient()

	info, err := client.GetQueueInfo(queueName)
	if err != nil {
		return err
	}

	fmt.Printf("Queue: %s\n", info.Queue)
	fmt.Printf("Paused: %v\n", info.Paused)
	fmt.Printf("Memory Usage: %d bytes\n", info.MemoryUsage)
	fmt.Println()
	fmt.Println("Task Counts:")
	fmt.Printf("  Pending:   %d\n", info.Pending)
	fmt.Printf("  Active:    %d\n", info.Active)
	fmt.Printf("  Scheduled: %d\n", info.Scheduled)
	fmt.Printf("  Retry:     %d\n", info.Retry)
	fmt.Printf("  Archived:  %d\n", info.Archived)
	fmt.Printf("  Completed: %d\n", info.Completed)

	return nil
}

func runQueuePause(cmd *cobra.Command, args []string) error {
	if err := validateArgs(args, "queue name"); err != nil {
		return err
	}
	queueName := args[0]

	// Remote mode: use API client
	if IsRemoteMode() {
		client := NewAPIClient()
		if err := client.PauseQueue(queueName); err != nil {
			return err
		}
		fmt.Printf("Queue '%s' paused successfully.\n", queueName)
		return nil
	}

	// Local mode: direct Redis access
	inspector, err := getInspector()
	if err != nil {
		return err
	}
	defer inspector.Close()

	if err := inspector.PauseQueue(queueName); err != nil {
		return fmt.Errorf("failed to pause queue: %w", err)
	}

	fmt.Printf("Queue '%s' paused successfully.\n", queueName)
	return nil
}

func runQueueUnpause(cmd *cobra.Command, args []string) error {
	if err := validateArgs(args, "queue name"); err != nil {
		return err
	}
	queueName := args[0]

	// Remote mode: use API client
	if IsRemoteMode() {
		client := NewAPIClient()
		if err := client.UnpauseQueue(queueName); err != nil {
			return err
		}
		fmt.Printf("Queue '%s' unpaused successfully.\n", queueName)
		return nil
	}

	// Local mode: direct Redis access
	inspector, err := getInspector()
	if err != nil {
		return err
	}
	defer inspector.Close()

	if err := inspector.UnpauseQueue(queueName); err != nil {
		return fmt.Errorf("failed to unpause queue: %w", err)
	}

	fmt.Printf("Queue '%s' unpaused successfully.\n", queueName)
	return nil
}
