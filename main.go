package main

import (
	"context"
	"flag"
	"github.com/SENERGY-Platform/senergy-load-test/pkg"
	"github.com/SENERGY-Platform/senergy-load-test/pkg/configuration"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	configLocation := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	config, err := configuration.LoadConfig(*configLocation)
	if err != nil {
		log.Fatal("ERROR: unable to load config ", err)
	}

	if config.IsCleanup {
		err = pkg.Cleanup(config)
		if err != nil {
			log.Fatal("ERROR:", err)
		}
	} else {
		wg := &sync.WaitGroup{}
		defer wg.Wait()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = pkg.Start(ctx, wg, config)
		if err != nil {
			log.Println("ERROR: ", err)
			return
		}

		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		sig := <-shutdown
		log.Println("received shutdown signal", sig)
	}
}
