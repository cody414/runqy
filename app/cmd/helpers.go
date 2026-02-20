package cmd

import (
	"fmt"
	"strings"
)

// validateArgs checks that positional arguments are non-empty after trimming whitespace.
// names is a list of human-readable names for each argument (e.g., "queue name", "task ID").
// It also trims whitespace from the args in-place.
func validateArgs(args []string, names ...string) error {
	for i, name := range names {
		if i < len(args) {
			args[i] = strings.TrimSpace(args[i])
			if args[i] == "" {
				return fmt.Errorf("%s cannot be empty", name)
			}
		}
	}
	return nil
}
