package service

import (
	"os"
	"os/signal"
	"syscall"
)

type Service interface {
	Init() error
	Start() error
	Stop() error
}

func Run(service Service, signals ...os.Signal) error {
	shutdown := make(chan struct{})

	if len(signals) == 0 {
		signals = append(signals, syscall.SIGINT, syscall.SIGTERM)
	}

	go func() {
		quitChan := make(chan os.Signal, 1)
		signal.Notify(quitChan, signals...)

		defer signal.Stop(quitChan)

		<-quitChan
		shutdown <- struct{}{}
	}()

	if err := service.Init(); err != nil {
		return err
	}

	if err := service.Start(); err != nil {
		return err
	}

	<-shutdown

	return service.Stop()
}
