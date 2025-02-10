package hello_mars

import (
	"context"
	"fmt"
)

type HelloMars struct{}

func NewHelloMars() *HelloMars {
	return &HelloMars{}
}

func (j *HelloMars) Init(ctx context.Context) error {
	fmt.Println("Hello Mars Init")
	return nil
}

func (j *HelloMars) Run(ctx context.Context) error {
	fmt.Println("Hello Mars Run")
	return nil
}

func (j *HelloMars) CleanUp(ctx context.Context) error {
	fmt.Println("Hello Mars CleanUp")
	return nil
}
