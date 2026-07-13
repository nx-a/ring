package ring

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	// DefaultMaxMemoryCache — лимит кеша логов в памяти по умолчанию (20 Мб)
	DefaultMaxMemoryCache int64 = 20 << 20
	// DefaultMaxFileCache — лимит файлового кеша логов по умолчанию (50 Мб)
	DefaultMaxFileCache int64 = 50 << 20
)

// CacheStats — состояние кеша логов.
type CacheStats struct {
	MemEntries int
	MemBytes   int64
	FileBytes  int64
}

// logCache кеширует логи, пока нет связи с сервером: сначала в памяти
// (не более maxMem байт), при переполнении сбрасывает в файл
// (не более maxFile байт, старые записи при переполнении удаляются).
// При восстановлении связи записи отдаются пачками через nextBatch,
// сначала самые старые из файла, затем из памяти.
type logCache struct {
	mu sync.Mutex

	path    string
	maxMem  int64
	maxFile int64

	mem      [][]byte
	memBytes int64

	spillSize int64

	replayF   *os.File
	replayR   *bufio.Reader
	replayPos int64

	lastFromFile   bool
	lastBatchBytes int64
}

// newLogCache создаёт кеш. key используется для формирования уникального
// имени spill-файла (например, адрес сервера).
func newLogCache(dir string, maxMem, maxFile int64, key string) (*logCache, error) {
	if maxMem <= 0 {
		maxMem = DefaultMaxMemoryCache
	}
	if maxFile <= 0 {
		maxFile = DefaultMaxFileCache
	}
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "ring-log-cache")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	sum := sha1.Sum([]byte(key))
	c := &logCache{
		path:    filepath.Join(dir, "ring-"+hex.EncodeToString(sum[:8])+".spill"),
		maxMem:  maxMem,
		maxFile: maxFile,
	}
	if st, err := os.Stat(c.path); err == nil {
		c.spillSize = st.Size()
	}
	return c, nil
}

// add добавляет запись в память; при превышении лимита памяти
// весь памятный кеш сбрасывается в файл.
func (c *logCache) add(entry []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]byte, len(entry))
	copy(cp, entry)
	c.mem = append(c.mem, cp)
	c.memBytes += int64(len(cp)) + 1
	if c.memBytes >= c.maxMem {
		c.spill()
	}
}

// hasPending сообщает, есть ли в кеше неотправленные логи.
func (c *logCache) hasPending() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.mem) > 0 || c.spillSize > 0
}

// stats возвращает текущее состояние кеша.
func (c *logCache) stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStats{
		MemEntries: len(c.mem),
		MemBytes:   c.memBytes,
		FileBytes:  c.spillSize,
	}
}

// nextBatch извлекает из кеша следующую порцию (до n записей): сначала
// самые старые из файла, затем из памяти. При неудачной отправке порцию
// необходимо вернуть через restore.
func (c *logCache) nextBatch(n int) [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFromFile = false
	c.lastBatchBytes = 0
	if n <= 0 {
		return nil
	}
	if c.spillSize > 0 && c.replayF == nil && !c.openReplay() {
		// файл пропал или пуст — забываем про него
		c.spillSize = 0
		c.replayPos = 0
	}
	if c.replayF != nil {
		out := c.readFileBatch(n)
		if len(out) > 0 {
			c.lastFromFile = true
			return out
		}
	}
	return c.popMemBatch(n)
}

// restore возвращает порцию, извлечённую nextBatch, обратно в кеш
// (вызывается при неудачной отправке, порядок записей сохраняется).
func (c *logCache) restore(batch [][]byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(batch) == 0 {
		return
	}
	if c.lastFromFile && c.replayF != nil {
		// откатываем позицию чтения файла на начало порции
		pos := c.replayPos - c.lastBatchBytes
		if pos < 0 {
			pos = 0
		}
		if _, err := c.replayF.Seek(pos, io.SeekStart); err == nil {
			c.replayR = bufio.NewReaderSize(c.replayF, 64*1024)
			c.replayPos = pos
			c.lastFromFile = false
			c.lastBatchBytes = 0
			return
		}
		c.closeReplayLocked()
		c.replayPos = 0
	}
	c.pushFrontMem(batch)
	c.lastFromFile = false
	c.lastBatchBytes = 0
}

// flushAll сбрасывает содержимое памяти в файл (вызывается при остановке).
func (c *logCache) flushAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.spill()
}

// closeReplay закрывает дескриптор файла воспроизведения, сохраняя позицию
// чтения (вызывается при разрыве соединения).
func (c *logCache) closeReplay() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeReplayLocked()
}

func (c *logCache) closeReplayLocked() {
	if c.replayF != nil {
		_ = c.replayF.Close()
		c.replayF = nil
		c.replayR = nil
	}
}

func (c *logCache) readFileBatch(n int) [][]byte {
	out := make([][]byte, 0, n)
	for len(out) < n {
		line, err := c.replayR.ReadBytes('\n')
		complete := len(line) > 0 && line[len(line)-1] == '\n'
		if complete {
			c.replayPos += int64(len(line))
			c.lastBatchBytes += int64(len(line))
			line = line[:len(line)-1]
			if len(line) > 0 {
				out = append(out, line)
			}
		}
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Log cache: replay read error: %v\n", err)
			} else if len(line) > 0 && !complete {
				// неполная строка в конце файла (аварийная остановка при записи)
				fmt.Printf("Log cache: dropping incomplete tail of spill file\n")
			}
			// во время воспроизведения файл не дополняется, поэтому EOF
			// означает, что все записи воспроизведены
			c.finishReplay()
			break
		}
	}
	return out
}

func (c *logCache) popMemBatch(n int) [][]byte {
	if len(c.mem) == 0 {
		return nil
	}
	k := min(n, len(c.mem))
	out := c.mem[:k]
	for _, e := range out {
		c.memBytes -= int64(len(e)) + 1
	}
	c.mem = c.mem[k:]
	if len(c.mem) == 0 {
		c.mem = nil
		c.memBytes = 0
	}
	return out
}

func (c *logCache) pushFrontMem(batch [][]byte) {
	for _, e := range batch {
		c.memBytes += int64(len(e)) + 1
	}
	c.mem = append(batch, c.mem...)
	if c.memBytes >= c.maxMem {
		c.spill()
	}
}

// spill сбрасывает весь памятный кеш в spill-файл.
func (c *logCache) spill() {
	if len(c.mem) == 0 {
		return
	}
	f, err := os.OpenFile(c.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("Log cache: failed to open spill file: %v, dropping %d entries\n", err, len(c.mem))
		c.resetMem()
		return
	}
	w := bufio.NewWriter(f)
	var written int64
	failed := false
	for _, e := range c.mem {
		n, err := w.Write(e)
		written += int64(n)
		if err != nil {
			failed = true
			break
		}
		if err := w.WriteByte('\n'); err != nil {
			failed = true
			break
		}
		written++
	}
	if err := w.Flush(); err != nil {
		failed = true
	}
	_ = f.Close()
	if failed {
		fmt.Printf("Log cache: failed to write spill file, some logs may be lost\n")
	}
	c.spillSize += written
	c.resetMem()
	if c.spillSize > c.maxFile {
		c.compact()
	}
}

func (c *logCache) resetMem() {
	c.mem = nil
	c.memBytes = 0
}

// compact обрезает spill-файл, оставляя только самые свежие записи,
// укладывающиеся в половину лимита.
func (c *logCache) compact() {
	if c.replayF != nil {
		c.closeReplayLocked()
		c.replayPos = 0
	}
	f, err := os.Open(c.path)
	if err != nil {
		fmt.Printf("Log cache: failed to open spill file for compaction: %v\n", err)
		return
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return
	}
	keep := c.maxFile / 2
	if st.Size() <= keep {
		_ = f.Close()
		return
	}
	start := st.Size() - keep
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		_ = f.Close()
		return
	}
	// выравниваем позицию по границе строки
	r := bufio.NewReaderSize(f, 64*1024)
	if _, err := r.ReadBytes('\n'); err != nil && err != io.EOF {
		_ = f.Close()
		return
	}
	tmp := c.path + ".tmp"
	tf, err := os.Create(tmp)
	if err != nil {
		_ = f.Close()
		fmt.Printf("Log cache: compaction failed: %v\n", err)
		return
	}
	copied, err := io.Copy(tf, r)
	_ = tf.Close()
	_ = f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		fmt.Printf("Log cache: compaction failed: %v\n", err)
		return
	}
	if err := os.Rename(tmp, c.path); err != nil {
		_ = os.Remove(tmp)
		fmt.Printf("Log cache: compaction rename failed: %v\n", err)
		return
	}
	c.spillSize = copied
	fmt.Printf("Log cache: spill file exceeded %d bytes, compacted to %d bytes, oldest logs dropped\n",
		c.maxFile, c.spillSize)
}

func (c *logCache) openReplay() bool {
	f, err := os.Open(c.path)
	if err != nil {
		return false
	}
	st, err := f.Stat()
	if err != nil || st.Size() == 0 {
		_ = f.Close()
		return false
	}
	c.spillSize = st.Size()
	if c.replayPos > 0 {
		if c.replayPos >= st.Size() {
			c.replayPos = 0
		} else if _, err := f.Seek(c.replayPos, io.SeekStart); err != nil {
			c.replayPos = 0
		}
	}
	c.replayF = f
	c.replayR = bufio.NewReaderSize(f, 64*1024)
	return true
}

func (c *logCache) finishReplay() {
	c.closeReplayLocked()
	c.replayPos = 0
	if err := os.Remove(c.path); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Log cache: failed to remove drained spill file: %v\n", err)
	}
	c.spillSize = 0
}
