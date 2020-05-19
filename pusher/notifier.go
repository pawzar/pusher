package pusher

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log"
	"sync"
	"time"
)

const (
	Verbose        = "verbose_logging"
	SkipEmptyLines = "skip_empty_lines"
)

type Pusher interface {
	Push(context.Context, io.Reader) error
}

func Push(ctx context.Context, client Pusher, input io.Reader, interval time.Duration) <-chan error {
	m, e := feed(ctx, input)
	return merge(e, process(ctx, client, m, interval))
}

func process(ctx context.Context, client Pusher, msgChan <-chan io.Reader, interval time.Duration) <-chan error {
	errChan := make(chan error)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		limiter := time.NewTicker(interval)
		defer limiter.Stop()

		for m := range msgChan {
			select {
			case <-ctx.Done():
				return
			case <-limiter.C:
				wg.Add(1)
				go func(msg io.Reader) {
					defer wg.Done()

					if err := client.Push(ctx, msg); err != nil {
						errChan <- err
					}
				}(m)
			}
		}
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	return errChan
}

func feed(ctx context.Context, input io.Reader) (<-chan io.Reader, <-chan error) {
	msgChan := make(chan io.Reader, 10)
	errChan := make(chan error, 1)
	var verbose bool
	if v := ctx.Value(Verbose); v != nil {
		verbose = v.(bool)
	}

	go func() {
		defer close(msgChan)
		defer close(errChan)

		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			b := scanner.Bytes()
			if verbose {
				log.Println("line:", string(b))
			}
			select {
			case <-ctx.Done():
				return
			case msgChan <- bytes.NewReader(b):
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	return msgChan, errChan
}

func merge(errs ...<-chan error) <-chan error {
	out := make(chan error)
	var wg sync.WaitGroup
	wg.Add(len(errs))
	for _, c := range errs {
		go func(c <-chan error) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
