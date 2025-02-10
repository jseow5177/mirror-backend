package main

import (
	"cdp/job/hello_mars"
	"cdp/job/hello_world"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

type Job interface {
	Init(ctx context.Context) error
	Run(ctx context.Context) error
	CleanUp(ctx context.Context) error
}

func main() {
	ctx := context.Background()

	jobs := map[string]Job{
		"hello_mars":  hello_mars.NewHelloMars(),
		"hello_world": hello_world.NewHelloWorld(),
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <job_name>")
		os.Exit(1)
	}

	jobName := os.Args[1]
	job, exists := jobs[jobName]
	if !exists {
		log.Ctx(ctx).Error().Msgf("job %s not found", jobName)
		os.Exit(1)
	}

	if err := job.Init(ctx); err != nil {
		log.Ctx(ctx).Error().Msgf("init job err: %v", err)
		os.Exit(1)
	}

	if err := job.Run(ctx); err != nil {
		log.Ctx(ctx).Error().Msgf("run job err: %v", err)
		os.Exit(1)
	}

	if err := job.CleanUp(ctx); err != nil {
		log.Ctx(ctx).Error().Msgf("cleanup job err: %v", err)
		os.Exit(1)
	}

	log.Ctx(ctx).Info().Msg("job executed successfully")
}
