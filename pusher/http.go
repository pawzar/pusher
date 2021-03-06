package pusher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func NewHttpPusher(url string) *HttpPusher {
	return &HttpPusher{u: url, c: &http.Client{
		Timeout: 5 * time.Second,
	}}
}

type HttpPusher struct {
	u string
	c *http.Client
}

func (c *HttpPusher) Push(ctx context.Context, message io.Reader) error {
	var body bytes.Buffer
	tee := io.TeeReader(message, &body)
	m, err := ioutil.ReadAll(tee)
	if err != nil {
		return &Error{Err: err}
	}
	verbose := flag(ctx, Verbose)
	if len(m) == 0 {
		if flag(ctx, SkipEmptyLines) && verbose {
			log.Println(SkipEmptyLines)
			return nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.u, &body)
	if err != nil {
		return &Error{Msg: string(m), Err: err}
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	if verbose {
		log.Printf("%s %s", req.Method, req.URL)
	}
	res, err := c.c.Do(req)
	if err != nil {
		return &Error{Msg: string(m), Err: err}
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusAccepted {
		return &Error{Msg: string(m), Err: fmt.Errorf("http return status code %d", res.StatusCode)}
	}

	return nil
}

func flag(ctx context.Context, k string) bool {
	if skip := ctx.Value(k); skip != nil {
		return skip.(bool)
	}
	return false
}
