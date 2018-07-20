package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
)

func interrupt() error {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return errors.Errorf("%s", <-c)
}
