package fetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestResumeCachesEmptyCompletedPart(t *testing.T) {
	calls := 0
	s := Session{Attempts: 1, Client: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		calls++
		if strings.HasSuffix(r.URL.Path, "bad") {
			return nil, io.ErrUnexpectedEOF
		}
		return response(""), nil
	})}}
	p := Part{Name: "empty", Endpoints: []string{"https://example.invalid/good"}, CachePath: filepath.Join(t.TempDir(), QueryKey([]byte("query")))}
	if _, err := s.Get(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	bad := p
	bad.CachePath = ""
	bad.Endpoints = []string{"https://example.invalid/bad"}
	if _, err := s.Get(context.Background(), bad); err == nil {
		t.Fatal("accepted interrupted part")
	}
	if _, err := s.Get(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("completed empty part fetched again: %d", calls)
	}
}

func TestDialTimeoutDisablesMirrorAcrossParts(t *testing.T) {
	bad := 0
	s := Session{Client: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "bad.invalid" {
			bad++
			return nil, &net.OpError{Op: "dial", Err: &net.DNSError{IsTimeout: true}}
		}
		return response("ok"), nil
	})}}
	for i := 0; i < 2; i++ {
		if _, err := s.Get(context.Background(), Part{Endpoints: []string{"https://bad.invalid", "https://good.invalid"}}); err != nil {
			t.Fatal(err)
		}
	}
	if bad != 1 {
		t.Fatalf("bad mirror retried %d times", bad)
	}
}

func TestDeadlineCancelsBackoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	s := Session{Backoff: time.Hour, Client: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF })}}
	_, err := s.Get(ctx, Part{Endpoints: []string{"https://example.invalid"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestRejectPartialAndOversizeBeforeCaching(t *testing.T) {
	for _, body := range []string{"partial", "too many bytes"} {
		t.Run(body, func(t *testing.T) {
			var log bytes.Buffer
			s := Session{Attempts: 1, Progress: &log, Client: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) { return response(body), nil })}}
			p := Part{Endpoints: []string{"https://example.invalid"}, CachePath: filepath.Join(t.TempDir(), "part"), MaxBytes: 8, Validate: func(b []byte) error {
				if string(b) != "complete" {
					return fmt.Errorf("partial response")
				}
				return nil
			}}
			if _, err := s.Get(context.Background(), p); err == nil {
				t.Fatal("accepted incomplete response")
			}
			if !strings.Contains(log.String(), "failure=") {
				t.Fatal("missing attempt diagnostic")
			}
			s.Client.Transport = transport(func(r *http.Request) (*http.Response, error) { return response("complete"), nil })
			if _, err := s.Get(context.Background(), p); err != nil {
				t.Fatal(err)
			}
		})
	}
}
