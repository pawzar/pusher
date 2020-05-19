package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"pusher/pusher"
)

func main() {
	config := configure()
	ctx := configuredContext(config)

	httpNotifier := pusher.NewHttpPusher(config.Url)
	errorChannel := pusher.Push(ctx, httpNotifier, os.Stdin, config.Interval)

	errCount := printErrors(errorChannel)
	if err := ctx.Err(); err != nil {
		log.Printf("ABORTED (%s) with %d errors", err, errCount)
	} else if errCount > 0 {
		log.Printf("DONE with %d errors", errCount)
	} else if config.Verbose {
		log.Println("DONE")
	}
}

func printErrors(errs <-chan error) int {
	i := 0
	for err := range errs {
		if errors.Is(err, pusher.NotificationError) {
			log.Println(err.(*pusher.Error).Msg, err)
		} else {
			log.Println(err)
		}
		i++
	}

	return i
}

func configuredContext(c config) context.Context {
	ctx := interruptable()
	if c.Verbose {
		ctx = context.WithValue(ctx, pusher.Verbose, true)
	}
	if c.SkipEmptyLines {
		ctx = context.WithValue(ctx, pusher.SkipEmptyLines, true)
	}
	return ctx
}

func interruptable() context.Context {
	return withCancelBy(context.Background(), os.Interrupt)
}

func withCancelBy(c context.Context, signals ...os.Signal) context.Context {
	if len(signals) == 0 {
		log.Fatalf("at least one signal type required")
	}

	ctx, cancel := context.WithCancel(c)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)

	go func() {
		<-ch
		cancel()
	}()

	return ctx
}

type config struct {
	Help           bool
	Verbose        bool
	SkipEmptyLines bool
	Url            string
	Interval       time.Duration
}

func configure() config {
	var c config

	flag.BoolVar(&c.Help, "h", false, "display usage")
	flag.BoolVar(&c.Verbose, "v", false, "verbose logging")
	flag.BoolVar(&c.SkipEmptyLines, "s", false, "skip empty lines")
	flag.StringVar(&c.Url, "t", "", "target URL for notifications")
	flag.DurationVar(&c.Interval, "i", time.Millisecond, "notification interval")

	flag.Parse()

	if c.Help || c.Url == "" {
		flag.Usage()
		os.Exit(1)
	}

	return c
}
