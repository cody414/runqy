package cmd

// Version information - set via ldflags at build time:
// go build -ldflags "-X github.com/Publikey/runqy/cmd.Version=1.0.0 -X github.com/Publikey/runqy/cmd.GitCommit=$(git rev-parse --short HEAD) -X github.com/Publikey/runqy/cmd.BuildDate=$(date -u +%Y-%m-%d)"
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)
