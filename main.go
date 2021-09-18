package main

import (
	"agent/api"
	"github.com/getsentry/sentry-go"
	"log"
	"time"
)

func main() {
	err := api.ReadConfig(&api.Settings)
	if err != nil {
		log.Fatalf("appSettings.read: %s", err)
	}

	err = sentry.Init(sentry.ClientOptions{
		Dsn: api.Settings.Sentry.Dsn,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	defer sentry.Flush(2 * time.Second)
	defer sentry.Recover()

	api.Run()
}
