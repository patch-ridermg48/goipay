package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/chekist32/goipay/internal/app"
)

func main() {
	configPath := flag.String("config", "config.yml", "Path to the config file")
	clientCAs := flag.String("client-ca", "", "Comma-separated list of paths to client certificate authority files (for mTLS)")
	flag.Parse()

	app := app.NewApp(app.CliOpts{
		ConfigPath:    *configPath,
		ClientCAPaths: *clientCAs,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
