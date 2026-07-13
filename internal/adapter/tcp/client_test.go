package tcp

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/nx-a/ring"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/dto"
)

type fakeDataService struct {
	mu     sync.Mutex
	writes []domain.Data
}

func (f *fakeDataService) Write(ctx context.Context, d *domain.Data) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes = append(f.writes, *d)
	return nil
}
func (f *fakeDataService) Find(ctx context.Context, d *dto.DataSelect) ([]domain.Data, error) {
	return nil, nil
}
func (f *fakeDataService) Clear()                                              {}
func (f *fakeDataService) Shutdown()                                           {}
func (f *fakeDataService) Count(ctx context.Context, id uint64) (int64, error) { return 0, nil }
func (f *fakeDataService) CountAll(ctx context.Context) (map[uint64]int64, error) {
	return nil, nil
}
func (f *fakeDataService) Subscribe() (ch <-chan domain.Data, id uint64) {
	return nil, 0
}
func (f *fakeDataService) Unsubscribe(id uint64) {}

func (f *fakeDataService) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.writes)
}

type fakeTokenService struct{}

func (f *fakeTokenService) GetByToken(token string) (map[string]any, error) {
	if token != "valid-token" {
		return nil, fmt.Errorf("invalid token")
	}
	return map[string]any{"bucketId": uint64(1), "bucket": "test"}, nil
}
func (f *fakeTokenService) Add(controlId uint64, token domain.Token) domain.Token {
	return token
}
func (f *fakeTokenService) GetByBucketId(controlId, id uint64) []domain.Token {
	return nil
}
func (f *fakeTokenService) Remove(controlId, id uint64) {}

// encodePayload кодирует сообщение так же, как клиентская библиотека:
// zlib + base64 + '\n'.
func encodePayload(t *testing.T, v any) []byte {
	t.Helper()
	payload, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out := make([]byte, base64.StdEncoding.EncodedLen(buf.Len()))
	base64.StdEncoding.Encode(out, buf.Bytes())
	return append(out, '\n')
}

func newPipeClient(t *testing.T, ds *fakeDataService) (net.Conn, *Client) {
	t.Helper()
	srvConn, cliConn := net.Pipe()
	c := NewClient(srvConn, ds, &fakeTokenService{})
	go c.Run()
	t.Cleanup(func() {
		_ = cliConn.Close()
		c.Close()
	})
	return cliConn, c
}

func sendAndAck(t *testing.T, conn net.Conn, payload []byte) {
	t.Helper()
	if _, err := conn.Write(payload); err != nil {
		t.Fatal(err)
	}
	ack, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if ack != "done\n" {
		t.Fatalf("expected ack 'done', got %q", ack)
	}
}

func TestHandleMessageBatch(t *testing.T) {
	ds := &fakeDataService{}
	conn, _ := newPipeClient(t, ds)

	t1 := "2024-01-02T03:04:05.123456789Z"
	t2 := "2024-01-02T03:04:06Z"
	entries := []ring.LogEntry{
		{Timestamp: t1, Level: "info", Message: "first", AppName: "app", Token: "valid-token"},
		{Timestamp: t2, Level: "error", Message: "second", AppName: "app", Token: "valid-token"},
	}
	sendAndAck(t, conn, encodePayload(t, entries))

	if ds.count() != 2 {
		t.Fatalf("expected 2 writes, got %d", ds.count())
	}
	for i, want := range []string{t1, t2} {
		got := ds.writes[i]
		wantTime, err := time.Parse(time.RFC3339Nano, want)
		if err != nil {
			t.Fatal(err)
		}
		if got.Time == nil || !got.Time.Equal(wantTime) {
			t.Fatalf("entry %d: expected time %v, got %v", i, wantTime, got.Time)
		}
		var fields map[string]any
		if err := json.Unmarshal(got.Val, &fields); err != nil {
			t.Fatal(err)
		}
		if fields["message"] != entries[i].Message {
			t.Fatalf("entry %d: expected message %q, got %q", i, entries[i].Message, fields["message"])
		}
		if got.Ext != "app" {
			t.Fatalf("entry %d: expected ext 'app', got %q", i, got.Ext)
		}
	}
}

func TestHandleMessageSingleEntry(t *testing.T) {
	ds := &fakeDataService{}
	conn, _ := newPipeClient(t, ds)

	entry := ring.LogEntry{
		Timestamp: "2024-05-06T07:08:09.5Z",
		Level:     "warn",
		Message:   "single",
		AppName:   "app",
		Token:     "valid-token",
	}
	sendAndAck(t, conn, encodePayload(t, entry))

	if ds.count() != 1 {
		t.Fatalf("expected 1 write, got %d", ds.count())
	}
	wantTime, _ := time.Parse(time.RFC3339Nano, entry.Timestamp)
	if !ds.writes[0].Time.Equal(wantTime) {
		t.Fatalf("expected time %v, got %v", wantTime, ds.writes[0].Time)
	}
}

func TestHandleMessageInvalidTokenSkipped(t *testing.T) {
	ds := &fakeDataService{}
	conn, _ := newPipeClient(t, ds)

	entries := []ring.LogEntry{
		{Timestamp: "2024-01-02T03:04:05Z", Level: "info", Message: "bad", AppName: "app", Token: "bad-token"},
		{Timestamp: "2024-01-02T03:04:06Z", Level: "info", Message: "good", AppName: "app", Token: "valid-token"},
	}
	sendAndAck(t, conn, encodePayload(t, entries))

	if ds.count() != 1 {
		t.Fatalf("expected 1 write (invalid token skipped), got %d", ds.count())
	}
	var fields map[string]any
	if err := json.Unmarshal(ds.writes[0].Val, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["message"] != "good" {
		t.Fatalf("expected message 'good', got %q", fields["message"])
	}
}

func TestHandleMessageInvalidTimestampFallsBackToNow(t *testing.T) {
	ds := &fakeDataService{}
	conn, _ := newPipeClient(t, ds)

	before := time.Now().Add(-time.Second)
	entry := ring.LogEntry{
		Timestamp: "not-a-date",
		Level:     "info",
		Message:   "no date",
		AppName:   "app",
		Token:     "valid-token",
	}
	sendAndAck(t, conn, encodePayload(t, entry))

	if ds.count() != 1 {
		t.Fatalf("expected 1 write, got %d", ds.count())
	}
	got := ds.writes[0].Time
	if got == nil || got.Before(before) || got.After(time.Now().Add(time.Second)) {
		t.Fatalf("expected fallback to current time, got %v", got)
	}
}
