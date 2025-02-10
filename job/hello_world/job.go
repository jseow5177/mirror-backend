package hello_world

import (
	"context"
	"fmt"
)

type HelloWorld struct{}

func NewHelloWorld() *HelloWorld {
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
