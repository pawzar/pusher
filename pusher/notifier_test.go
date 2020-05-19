package pusher

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNotify(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		notifier Pusher
		input    io.Reader
		interval time.Duration
		want     error
	}{
		{
			name:     "single message",
			input:    strings.NewReader("message1"),
			notifier: newCountingNotifier(1),
		},
		{
			name:     "2 messages",
			input:    strings.NewReader("message1\nmessage2"),
			notifier: newCountingNotifier(2),
		},
		{
			name:     "9 messages (one of them is empty",
			input:    strings.NewReader("a\nb\nc\n\nd\ne\nf\ng\nh"),
			notifier: newCountingNotifier(9),
		},
		{
			name:     "bad reader",
			input:    badReader("badReader"),
			want:     errors.New("badReader"),
			notifier: newCountingNotifier(0),
		},
		{
			name:     "cancelled context",
			input:    strings.NewReader("message1"),
			ctx:      cancelledContext(),
			notifier: newCountingNotifier(0),
		},
	}
	for _, tt := range tests {
		if tt.interval == 0 {
			tt.interval = 1
		}
		ctx := tt.ctx
		if ctx == nil {
			ctx = context.TODO()
		}

		t.Run(tt.name, func(t *testing.T) {
			if err := <-Push(ctx, tt.notifier, tt.input, tt.interval); tt.want == nil && err != nil || tt.want != nil && err == nil || tt.want != nil && err != nil && err.Error() != tt.want.Error() {
				t.Errorf("<-Notify() = %v , want %v", err, tt.want)
			}
			if notifier, ok := tt.notifier.(*countingNotifier); ok && !notifier.OK() {
				t.Errorf("notifier called = %d times, want %d", notifier.counter, notifier.expectation)
			}
		})
	}
}

func ctxWithTimeout(d time.Duration) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), d)
	return ctx
}

func Test_process(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		notifier Pusher
		input    <-chan io.Reader
		interval time.Duration
		want     error
	}{
		{
			name:     "error on bad reader",
			input:    inputWith(badReader("badReader")),
			notifier: getTestNotifier(testCase{retCode: http.StatusOK}),
			want:     errors.New("badReader"),
		},
		{
			name:     "no error when ok",
			notifier: getTestNotifier(testCase{retCode: http.StatusOK}),
		},
		{
			name:     "error when bad request",
			notifier: getTestNotifier(testCase{retCode: http.StatusBadRequest}),
			want:     errors.New(`http return status code 400`),
		},
		{
			name:     "skip for context cancelled before processing",
			ctx:      cancelledContext(),
			notifier: getTestNotifier(testCase{}),
		},
		{
			name:     "error for context cancelled while in progress",
			input:    inputWith(newSlowReader("msg", time.Millisecond)),
			ctx:      ctxWithTimeout(time.Millisecond),
			notifier: getTestNotifier(testCase{}),
			want:     errors.New(`Post "http://localhost:8080": context deadline exceeded`),
		},
	}
	for _, tt := range tests {
		if tt.interval == 0 {
			tt.interval = 1
		}
		ctx := tt.ctx
		if ctx == nil {
			ctx = verboseContext()
		}
		input := tt.input
		if input == nil {
			input = inputWith(strings.NewReader(""))
		}

		t.Run(tt.name, func(t *testing.T) {
			if err := <-process(ctx, tt.notifier, input, tt.interval); tt.want == nil && err != nil || tt.want != nil && err == nil || tt.want != nil && err != nil && err.Error() != tt.want.Error() {
				t.Errorf("<-process() = %v , want %v", err, tt.want)
			}
		})
	}
}

type countingNotifier struct {
	counter     int
	expectation int
}

func (c *countingNotifier) OK() bool {
	return c.counter == c.expectation
}
func (c *countingNotifier) Push(context.Context, io.Reader) error {
	c.counter++
	return nil
}

func newCountingNotifier(i int) *countingNotifier {
	return &countingNotifier{expectation: i}
}

type slowReader struct {
	r io.Reader
	d time.Duration
	o sync.Once
}

func newSlowReader(s string, d time.Duration) *slowReader {
	return &slowReader{r: strings.NewReader(s), d: d}
}

func (s *slowReader) Read(b []byte) (int, error) {
	s.o.Do(func() {
		time.Sleep(s.d)
	})
	return s.r.Read(b)
}

func inputWith(r ...io.Reader) <-chan io.Reader {
	ch := make(chan io.Reader, len(r))
	for _, rr := range r {
		ch <- rr
	}
	close(ch)
	return ch
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(verboseContext())
	cancel()
	return ctx
}

func verboseContext() context.Context {
	return context.WithValue(context.Background(), Verbose, true)
}
