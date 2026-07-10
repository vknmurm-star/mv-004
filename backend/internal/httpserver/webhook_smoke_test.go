package httpserver

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// verify recentSet dedup semantics in isolation (the MAX webhook uses it to
// guard against redeliveries of the same message_id).
func TestRecentSet_DeduplicatesKey(t *testing.T) {
	s := newRecentSet(8)
	if !s.mark("a") {
		t.Fatal("first 'a' should be marked true")
	}
	if s.mark("a") {
		t.Fatal("second 'a' should be marked false (duplicate)")
	}
	if !s.mark("b") {
		t.Fatal("first 'b' should be true")
	}
}

func TestRecentSet_BoundedEviction(t *testing.T) {
	s := newRecentSet(2)
	s.mark("a")
	s.mark("b")
	// third insert triggers eviction path; previously-seen key may resurface.
	s.mark("c")
	// Regardless of which key was evicted, we should not crash and should
	// still accept a fresh mark.
	if !s.mark("z") {
		t.Fatal("fresh key 'z' should be accepted after eviction")
	}
}

func TestRecentSet_ThreadSafe(t *testing.T) {
	s := newRecentSet(1024)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = s.mark("dup")
			_ = s.mark(itoaSimple(i))
		}(i)
	}
	wg.Wait()
}

// smoke assertion that the public Health handler imports a DB-shaped Deps
// without crashing the package — kept lightweight; integration against a
// real DB lives in cmd/tests.
func TestHandlers_Construct(t *testing.T) {
	h := &handlers{d: &Deps{}, seatStore: newSeatStore(), recentInbound: newRecentSet(8)}
	if h == nil || h.recentInbound == nil {
		t.Fatal("handler wiring failed")
	}
	// recentInbound dedup
	if !h.recentInbound.mark("m1") {
		t.Fatal("m1 should accept first time")
	}
}

// small helper mirroring strconv.Itoa just to keep the test dep-free.
func itoaSimple(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// guard against accidental behaviour changes in WriteJSON signature by
// exercising the readAllWithLimit cap with a body larger than the limit.
func TestReadAllWithLimit_CapsSize(t *testing.T) {
	body := make([]byte, 1<<16)
	for i := range body {
		body[i] = 'x'
	}
	r := httptest.NewRequest(http.MethodPost, "/", bytesReader(body))
	got, err := readAllWithLimit(r, 1<<10)
	if err != nil {
		t.Fatalf("readAllWithLimit err: %v", err)
	}
	if len(got) != 1<<10 {
		t.Errorf("limit not enforced: got %d bytes, want %d", len(got), 1<<10)
	}
}

// tiny helper to satisfy io.Reader needs without importing bytes here.
func bytesReader(b []byte) *bytesReaderT { return &bytesReaderT{b: b} }

type bytesReaderT struct {
	b []byte
	i int
}

func (r *bytesReaderT) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, errEOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

var errEOF = errEOFType{}

type errEOFType struct{}

func (errEOFType) Error() string { return "EOF" }
