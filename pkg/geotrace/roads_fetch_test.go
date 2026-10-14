package geotrace

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roadTestTransport func(*http.Request) (*http.Response, error)

func (f roadTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRoadFetchDeadlineCancelsRequest(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roadTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	_, err := fetchOverpassRoads("osm-interstate", Options{BBox: BBox{0, 0, 1, 1}, Client: client, RoadsTimeout: 5 * time.Millisecond})
	if !errors.Is(err, context.DeadlineExceeded) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestRoadFetchProgressAndQueryProvenance(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roadTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"elements":[]}`)), Header: http.Header{}}, nil
	})}
	var progress bytes.Buffer
	raw, err := fetchOverpassRoads("osm-interstate", Options{BBox: BBox{0, 0, 6, 6}, Client: client, Progress: &progress})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 36 || !strings.Contains(progress.String(), "chunk 36/36 ways=0") || !strings.Contains(progress.String(), "endpoint=") || !bytes.Contains(raw, []byte(`"kit_query_bbox"`)) {
		t.Fatalf("calls=%d progress=%s body=%s", calls, progress.String(), raw)
	}
}

func TestRoadFetchRejectsPartialSuccess(t *testing.T) {
	client := &http.Client{Transport: roadTestTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"elements":[],"remark":"runtime error: timeout"}`)), Header: http.Header{}}, nil
	})}
	if _, err := fetchOverpassRoads("osm-interstate", Options{BBox: BBox{0, 0, 1, 1}, Client: client}); err == nil {
		t.Fatal("partial provider response accepted")
	}
}

func TestRoadFetchLogsAttemptFailure(t *testing.T) {
	var log bytes.Buffer
	calls := 0
	client := &http.Client{Transport: roadTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("rate limited")), Header: http.Header{}}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"elements":[]}`)), Header: http.Header{}}, nil
	})}
	_, err := fetchOverpassRoads("osm-interstate", Options{BBox: BBox{0, 0, 1, 1}, Client: client, Progress: &log})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log.String(), `failure="HTTP 429: rate limited"`) || !strings.Contains(log.String(), "attempt=2") {
		t.Fatalf("missing failure detail: %s", log.String())
	}
}

func TestRoadFetchResumesCompletedChunks(t *testing.T) {
	calls := 0
	opts := Options{BBox: BBox{0, 0, 6, 6}, RoadsCacheDir: t.TempDir()}
	opts.Client = &http.Client{Transport: roadTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls > 1 {
			return nil, io.ErrUnexpectedEOF
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"elements":[]}`)), Header: http.Header{}}, nil
	})}
	if _, err := fetchOverpassRoads("osm-interstate", opts); err == nil {
		t.Fatal("expected interrupted fetch")
	}
	calls = 0
	opts.Client = &http.Client{Transport: roadTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"elements":[]}`)), Header: http.Header{}}, nil
	})}
	if _, err := fetchOverpassRoads("osm-interstate", opts); err != nil {
		t.Fatal(err)
	}
	if calls != 35 {
		t.Fatalf("retried completed chunk: %d requests", calls)
	}
	calls = 0
	if _, err := fetchOverpassRoads("osm-interstate", opts); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("complete cache not reused")
	}
}

func TestRoadFetchSkipsTimedOutMirrorForRemainingChunks(t *testing.T) {
	bad := 0
	client := &http.Client{Transport: roadTestTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "overpass-api.de" {
			bad++
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "timeout", IsTimeout: true}}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"elements":[]}`)), Header: http.Header{}}, nil
	})}
	if _, err := fetchOverpassRoads("osm-interstate", Options{BBox: BBox{0, 0, 6, 6}, Client: client}); err != nil {
		t.Fatal(err)
	}
	if bad != 1 {
		t.Fatalf("timed-out mirror retried %d times", bad)
	}
}
