package ring

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

func entry(n int) []byte {
	return []byte(fmt.Sprintf(`{"n":%d}`, n))
}

// drain извлекает все записи из кеша пачками по size.
func drain(t *testing.T, c *logCache, size int) [][]byte {
	t.Helper()
	var got [][]byte
	for c.hasPending() {
		b := c.nextBatch(size)
		if len(b) == 0 {
			t.Fatal("nextBatch returned empty batch while cache has pending entries")
		}
		got = append(got, b...)
	}
	return got
}

func assertOrder(t *testing.T, got [][]byte, want int) {
	t.Helper()
	if len(got) != want {
		t.Fatalf("expected %d entries, got %d", want, len(got))
	}
	for i, b := range got {
		if !bytes.Equal(b, entry(i)) {
			t.Fatalf("entry %d: expected %s, got %s", i, entry(i), b)
		}
	}
}

func TestCacheMemoryReplay(t *testing.T) {
	c, err := newLogCache(t.TempDir(), 1024, 4096, "mem")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		c.add(entry(i))
	}
	if !c.hasPending() {
		t.Fatal("expected pending entries")
	}
	got := drain(t, c, 3)
	assertOrder(t, got, 5)
	if _, err := os.Stat(c.path); !os.IsNotExist(err) {
		t.Fatal("spill file must not be created when memory limit is not exceeded")
	}
}

func TestCacheSpillAndReplayOrder(t *testing.T) {
	// лимит памяти 64 байта: ~8 записей по 8 байт вызывают сброс в файл
	c, err := newLogCache(t.TempDir(), 64, 1<<20, "spill")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		c.add(entry(i))
	}
	if c.stats().FileBytes == 0 {
		t.Fatal("expected spill to file")
	}
	got := drain(t, c, 3)
	assertOrder(t, got, 20)
	if _, err := os.Stat(c.path); !os.IsNotExist(err) {
		t.Fatal("spill file must be removed after full replay")
	}
}

func TestCacheRestoreFileBatch(t *testing.T) {
	c, err := newLogCache(t.TempDir(), 1024, 1<<20, "restore-file")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		c.add(entry(i))
	}
	c.flushAll()

	first := c.nextBatch(4)
	if len(first) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(first))
	}
	c.restore(first)
	again := c.nextBatch(4)
	if len(again) != 4 {
		t.Fatalf("expected 4 entries after restore, got %d", len(again))
	}
	for i := range first {
		if !bytes.Equal(first[i], again[i]) {
			t.Fatalf("entry %d differs after restore: %s != %s", i, first[i], again[i])
		}
	}
}

func TestCacheRestoreMemBatch(t *testing.T) {
	c, err := newLogCache(t.TempDir(), 1024, 1<<20, "restore-mem")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		c.add(entry(i))
	}
	first := c.nextBatch(2)
	c.restore(first)
	got := drain(t, c, 10)
	assertOrder(t, got, 5)
}

func TestCacheReplayPositionKeptOnDisconnect(t *testing.T) {
	c, err := newLogCache(t.TempDir(), 1024, 1<<20, "pos")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		c.add(entry(i))
	}
	c.flushAll()

	first := c.nextBatch(3)
	if len(first) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(first))
	}
	// имитируем разрыв соединения: дескриптор закрывается, позиция сохраняется
	c.closeReplay()
	got := drain(t, c, 3)
	if len(got) != 7 {
		t.Fatalf("expected remaining 7 entries, got %d: %v", len(got), got)
	}
	for i, b := range got {
		if !bytes.Equal(b, entry(i+3)) {
			t.Fatalf("entry %d: expected %s, got %s", i, entry(i+3), b)
		}
	}
}

func TestCacheCompact(t *testing.T) {
	// лимит файла 128 байт: при переполнении остаются только самые свежие записи
	c, err := newLogCache(t.TempDir(), 64, 128, "compact")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		c.add(entry(i))
	}
	c.flushAll()
	if st := c.stats(); st.FileBytes > c.maxFile {
		t.Fatalf("spill size %d exceeds file limit %d", st.FileBytes, c.maxFile)
	}
	data, err := os.ReadFile(c.path)
	if err != nil {
		t.Fatal(err)
	}
	// файл должен содержать только целые строки
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	prev := -1
	for _, line := range lines {
		if !strings.HasPrefix(line, `{"n":`) || !strings.HasSuffix(line, "}") {
			t.Fatalf("corrupted line in spill file: %q", line)
		}
		n, err := strconv.Atoi(line[5 : len(line)-1])
		if err != nil {
			t.Fatalf("bad entry %q: %v", line, err)
		}
		if n <= prev {
			t.Fatalf("entries out of order: %d after %d", n, prev)
		}
		prev = n
	}
	// после всего — оставшиеся записи должны быть самыми свежими
	got := drain(t, c, 2)
	want := 100 - len(lines)
	if len(got) != 100-want {
		t.Fatalf("expected %d entries total, got %d", 100-want, len(got))
	}
	for i, b := range got {
		if !bytes.Equal(b, entry(want+i)) {
			t.Fatalf("entry %d: expected %s, got %s", i, entry(want+i), b)
		}
	}
}

func TestCacheFlushAllAndPersistence(t *testing.T) {
	dir := t.TempDir()
	c1, err := newLogCache(dir, 1024, 1<<20, "persist")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		c1.add(entry(i))
	}
	c1.flushAll()

	// новый инстанс (перезапуск приложения) подхватывает spill-файл
	c2, err := newLogCache(dir, 1024, 1<<20, "persist")
	if err != nil {
		t.Fatal(err)
	}
	if !c2.hasPending() {
		t.Fatal("expected pending entries after restart")
	}
	got := drain(t, c2, 2)
	assertOrder(t, got, 5)
}

func TestCacheSpillFileIsolatedByKey(t *testing.T) {
	dir := t.TempDir()
	c1, err := newLogCache(dir, 1024, 1<<20, "server-a:7777")
	if err != nil {
		t.Fatal(err)
	}
	c2, err := newLogCache(dir, 1024, 1<<20, "server-b:7777")
	if err != nil {
		t.Fatal(err)
	}
	if c1.path == c2.path {
		t.Fatal("spill files for different servers must be isolated")
	}
}
