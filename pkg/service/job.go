package service

import "context"

type Job interface {
	Init(ctx context.Context) error
	Run(ctx context.Context) error
	CleanUp(ctx context.Context) error
}
