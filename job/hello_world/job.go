package hello_world

import (
	"cdp/pkg/service"
	"context"
	"fmt"
)

type HelloWorld struct{}

func New() service.Job {
	return &HelloWorld{}
}

func (j *HelloWorld) Init(ctx context.Context) error {
	fmt.Println("Hello World Init")
	return nil
}

func (j *HelloWorld) Run(ctx context.Context) error {
	fmt.Println("Hello World Run")
	return nil
}

func (j *HelloWorld) CleanUp(ctx context.Context) error {
	fmt.Println("Hello World CleanUp")
	return nil
}
