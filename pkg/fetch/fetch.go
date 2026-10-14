// Package fetch provides bounded, resumable downloads. Callers own payload
// validation and provenance formats; a successful HTTP status alone is not proof
// that a provider returned a complete result.
package fetch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session is sequential: its disabled-mirror set lasts for the entire job.
// Supply one context with a deadline for the whole multi-part job.
type Session struct {
	Client   *http.Client
	Progress io.Writer
	Attempts int
	Backoff  time.Duration
	disabled map[string]bool
}

type Part struct {
	Name             string
	Endpoints        []string
	Method           string
	Body             []byte
	Header           http.Header
	MaxBytes         int64
	CachePath        string
	Validate         func([]byte) error
	ValidateResponse func(*http.Response) error
}

type StatusError struct {
	Code   int
	Detail string
}

func (e *StatusError) Error() string { return fmt.Sprintf("HTTP %d: %s", e.Code, e.Detail) }

func QueryKey(query []byte) string {
	sum := sha256.Sum256(query)
	return hex.EncodeToString(sum[:])
}

// AtomicWrite never publishes an unfinished part. Empty successful parts are valid.
func AtomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".fetch-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func (s *Session) log(format string, args ...any) {
	if s.Progress != nil {
		fmt.Fprintf(s.Progress, format, args...)
	}
}

func (s *Session) Get(ctx context.Context, p Part) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit := p.MaxBytes
	if limit <= 0 {
		limit = 64 << 20
	}
	valid := func(b []byte) error {
		if int64(len(b)) > limit {
			return fmt.Errorf("response exceeds %d bytes", limit)
		}
		if p.Validate != nil {
			return p.Validate(b)
		}
		return nil
	}
	if p.CachePath != "" {
		if b, err := os.ReadFile(p.CachePath); err == nil && valid(b) == nil {
			s.log("%s cached=true bytes=%d\n", p.Name, len(b))
			return b, nil
		}
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	attempts := s.Attempts
	if attempts <= 0 {
		attempts = 2
	}
	backoff := s.Backoff
	if backoff <= 0 {
		backoff = time.Second
	}
	if s.disabled == nil {
		s.disabled = map[string]bool{}
	}
	var last error
	for _, endpoint := range p.Endpoints {
		if s.disabled[endpoint] {
			continue
		}
		for attempt := 1; attempt <= attempts; attempt++ {
			if attempt > 1 {
				timer := time.NewTimer(backoff)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
				}
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			started := time.Now()
			s.log("%s endpoint=%s attempt=%d\n", p.Name, endpoint, attempt)
			req, err := http.NewRequestWithContext(ctx, p.Method, endpoint, bytes.NewReader(p.Body))
			if err != nil {
				return nil, err
			}
			req.Header = p.Header.Clone()
			resp, err := client.Do(req)
			var data []byte
			if err == nil {
				data, err = io.ReadAll(io.LimitReader(resp.Body, limit+1))
				closeErr := resp.Body.Close()
				if err == nil {
					err = closeErr
				}
				if err == nil && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
					detail := data
					if len(detail) > 500 {
						detail = detail[:500]
					}
					err = &StatusError{resp.StatusCode, strings.TrimSpace(string(detail))}
				}
				if err == nil && p.ValidateResponse != nil {
					err = p.ValidateResponse(resp)
				}
				if err == nil {
					err = valid(data)
				}
			}
			if err == nil {
				if p.CachePath != "" {
					if err = AtomicWrite(p.CachePath, data); err != nil {
						return nil, err
					}
				}
				s.log("%s complete bytes=%d elapsed=%s\n", p.Name, len(data), time.Since(started).Round(time.Millisecond))
				return data, nil
			}
			last = err
			s.log("%s endpoint=%s attempt=%d elapsed=%s failure=%q\n", p.Name, endpoint, attempt, time.Since(started).Round(time.Millisecond), err.Error())
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			var op *net.OpError
			if errors.As(err, &op) && op.Op == "dial" && op.Timeout() {
				s.disabled[endpoint] = true
				s.log("%s endpoint=%s disabled after dial timeout\n", p.Name, endpoint)
				break
			}
			var status *StatusError
			if errors.As(err, &status) && status.Code == http.StatusNotFound {
				break
			}
		}
	}
	if last == nil {
		last = fmt.Errorf("no reachable endpoints remain")
	}
	return nil, last
}
