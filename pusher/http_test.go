package pusher

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type testCase struct {
	name    string
	ctx     context.Context
	message io.Reader
	retCode int
	url     string
	want    error
}

func TestHttpClient_Notify(t *testing.T) {
	tests := []testCase{
		{
			name:    "error for bad reader",
			message: badReader("bad-reader"),
			want:    errors.New("bad-reader"),
		},
		{
			name:    "do not skip empty line",
			retCode: http.StatusBadRequest,                     // ensure error
			want:    errors.New(`http return status code 400`), // expect error when not skipped
		},
		{
			name:    "skip empty line",
			ctx:     context.WithValue(context.WithValue(context.TODO(), Verbose, true), SkipEmptyLines, true),
			retCode: http.StatusBadRequest, // ensure error if not skipped
			want:    nil,                   // expect no error because should be skipped
		},
		{
			name:    "no error status code 200",
			retCode: http.StatusOK,
		},
		{
			name:    "no error status code 201",
			retCode: http.StatusCreated,
		},
		{
			name:    "no error status code 202",
			retCode: http.StatusAccepted,
		},
		{
			name: "error for cancelled context",
			ctx:  cancelledContext(),
			want: errors.New(`Post "http://localhost:8080": context canceled`),
		},
		{
			name:    "error status code 400",
			retCode: http.StatusBadRequest,
			want:    errors.New(`http return status code 400`),
		},
		{
			name: "error for bad url",
			url:  "http://localhost:80xx",
			want: errors.New(`parse "http://localhost:80xx": invalid port ":80xx" after host`),
		},
		{
			name:    "error status code 302 without location",
			retCode: http.StatusFound,
			want:    errors.New(`Post "": 302 response missing Location header`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := getTestNotifier(tt)

			ctx := tt.ctx
			if ctx == nil {
				ctx = context.TODO()
			}
			message := tt.message
			if message == nil {
				message = strings.NewReader("")
			}

			err := n.Push(ctx, message)

			if tt.want == nil && err != nil || tt.want != nil && err == nil || tt.want != nil && err != nil && err.Error() != tt.want.Error() {
				t.Errorf("Notify() = %v , want %v", err, tt.want)
			}

			if err != nil && !errors.Is(err, NotificationError) {
				t.Errorf("Notify() [error type] = %T , want NotificationError", err)
			}
		})
	}
}

func getTestNotifier(tc testCase) Pusher {
	if tc.retCode == 0 {
		if tc.url == "" {
			return NewHttpPusher("http://localhost:8080")
		}
		return NewHttpPusher(tc.url)
	}

	return &HttpPusher{u: tc.url, c: funcHttpClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: tc.retCode,
			Header:     make(http.Header),
		}, nil
	})}
}

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func funcHttpClient(f RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: f,
	}
}

type badReader string

func (b badReader) Read([]byte) (int, error) {
	return 0, errors.New(string(b))
}
