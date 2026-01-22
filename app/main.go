package main

import (
	"github.com/Publikey/runqy/cmd"
	_ "github.com/Publikey/runqy/docs"
)

//	@title			Runqy Queue API
//	@version		1.0.0
//	@description	Task queueing service built on asynq for managing distributed task processing

func main() {
	cmd.Execute()
}
