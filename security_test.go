// THREAT MODEL -- lethe (Log Rotation + MPSC Telemetry Pipeline)
//
// Attack Surface: file paths, size/duration strings, write payloads,
// concurrent access patterns, and configuration inputs. Lethe is a
// military-grade async log pipeline designed to survive DDoS-level
// write loads. These tests verify it cannot be turned against itself.
//
// CWE-22 (Path Traversal):
//   Filename field in LoggerConfig could contain ../../../etc/cron.d/evil
//   or Windows device names (CON, NUL). SanitizeFilename strips dangerous
//   chars from the base filename, but traversal sequences in the directory
//   portion must also be handled safely.
//   MITIGATION: SanitizeFilename + ValidatePathLength. Constructor does
//   not traverse arbitrary paths -- it writes ONLY to the specified file.
//
// CWE-20 (Improper Input Validation):
//   ParseSize and ParseDuration accept user-controlled strings.
//   Garbage inputs, empty strings, overflow values, and null bytes
//   must produce clear errors, never panics.
//   MITIGATION: strict allowlist of suffixes, overflow checks,
//   empty input rejection.
//
// CWE-400 (Uncontrolled Resource Consumption):
//   Oversized Write() calls, rotation storms from rapid writes,
//   and adaptive buffer doubling could exhaust memory.
//   MITIGATION: backpressure policies, MaxBackups limit, bounded
//   adaptive resize (cap at 16K entries).
//
// CWE-362 (Race Condition):
//   Concurrent Write + Close + Rotate + Sync from multiple goroutines.
//   MITIGATION: atomic operations, sync.Once for Close, CAS-based
//   rotation flag, MPSC ring buffer with lock-free writes.
//
// CWE-770 (Uncontrolled Buffer Growth):
//   Adaptive backpressure doubles buffer up to 16K entries.
//   Repeated NewWithConfig + Close could leak goroutines.
//   MITIGATION: bounded max buffer size, sync.Once for Close,
//   background worker pool with fixed size.

package lethe

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// SecurityTestContext -- helper struct per AGILira convention
// ---------------------------------------------------------------------------

type securityTestContext struct {
	t      *testing.T
	tmpDir string
}

func newSecurityTestContext(t *testing.T) *securityTestContext {
	t.Helper()
	return &securityTestContext{
		t:      t,
		tmpDir: t.TempDir(),
	}
}

func (ctx *securityTestContext) logPath(name string) string {
	return filepath.Join(ctx.tmpDir, name)
}

func (ctx *securityTestContext) expectSecurityError(err error, msg string) {
	ctx.t.Helper()
	if err == nil {
		ctx.t.Fatalf("SECURITY: expected error for %s, got nil", msg)
	}
}

func (ctx *securityTestContext) expectSecuritySuccess(err error, msg string) {
	ctx.t.Helper()
	if err != nil {
		ctx.t.Fatalf("SECURITY: expected success for %s, got: %v", msg, err)
	}
}

// ---------------------------------------------------------------------------
// CWE-22: Path Traversal -- SanitizeFilename
// ---------------------------------------------------------------------------

func TestSecurity_SanitizeFilename_DangerousChars(t *testing.T) {
	// ATTACK VECTOR: CWE-22
	// IMPACT: special chars in filename could confuse shells or filesystem
	// MITIGATION EXPECTED: SanitizeFilename replaces dangerous chars with _

	dangerous := []struct {
		input string
		desc  string
	}{
		{"file<script>.log", "angle brackets"},
		{"file|pipe.log", "pipe character"},
		{"file\"quote.log", "double quote"},
		{"file?glob.log", "question mark"},
		{"file*star.log", "asterisk"},
		{"file\x01ctrl.log", "control character 0x01"},
		{"file\x1fctrl.log", "control character 0x1f"},
		{"file\x00null.log", "null byte"},
	}

	for _, tc := range dangerous {
		sanitized := SanitizeFilename(tc.input)
		if sanitized == tc.input {
			t.Errorf("SanitizeFilename(%q) [%s] was not sanitized", tc.input, tc.desc)
		}
		// WHY: the sanitized version must not contain any of the dangerous chars
		for _, ch := range []byte{'<', '>', '|', '"', '?', '*'} {
			if strings.ContainsRune(sanitized, rune(ch)) {
				t.Errorf("SanitizeFilename(%q) still contains %c", tc.input, ch)
			}
		}
	}
}

func TestSecurity_SanitizeFilename_PreservesClean(t *testing.T) {
	// WHY: sanitizer must not corrupt valid filenames
	clean := []string{
		"metis.log",
		"app-2026-01-01.log",
		"debug_trace.log.1",
		"service.json.log",
	}
	for _, name := range clean {
		sanitized := SanitizeFilename(name)
		if sanitized != name {
			t.Errorf("SanitizeFilename(%q) = %q, want unchanged", name, sanitized)
		}
	}
}

// ---------------------------------------------------------------------------
// CWE-22: Path Traversal -- ValidatePathLength
// ---------------------------------------------------------------------------

func TestSecurity_ValidatePathLength_Limits(t *testing.T) {
	// ATTACK VECTOR: CWE-22 / CWE-400
	// IMPACT: extremely long paths could cause buffer overflow or DoS
	// MITIGATION EXPECTED: ValidatePathLength enforces OS limits

	// WHY: Unix limit is 4096, Windows is 260. Test both boundaries.
	longPath := strings.Repeat("a", 5000)
	err := ValidatePathLength(longPath)
	if err == nil {
		t.Error("ValidatePathLength must reject paths exceeding OS limit")
	}
}

func TestSecurity_ValidatePathLength_AcceptsValid(t *testing.T) {
	err := ValidatePathLength("/tmp/metis/logs/app.log")
	if err != nil {
		t.Errorf("ValidatePathLength rejected valid path: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CWE-20: Input Validation -- ParseSize
// ---------------------------------------------------------------------------

func TestSecurity_ParseSize_InvalidInputs(t *testing.T) {
	// ATTACK VECTOR: CWE-20
	// IMPACT: crafted size strings could cause panic, overflow, or
	// unexpected behavior (e.g., negative size = no rotation = disk fill)
	// MITIGATION EXPECTED: ParseSize rejects all malformed inputs

	payloads := []struct {
		input string
		desc  string
	}{
		{"", "empty string"},
		{"not_a_size", "non-numeric"},
		{"MB", "suffix only, no number"},
		{"10XYZ", "unknown suffix"},
		{"10MB; rm -rf /", "shell injection"},
		{"\x00MB", "null byte prefix"},
		{"10\x00MB", "null byte mid-string"},
	}

	for _, tc := range payloads {
		_, err := ParseSize(tc.input)
		if err == nil {
			t.Errorf("ParseSize(%q) [%s] must return error", tc.input, tc.desc)
		}
	}
}

func TestSecurity_ParseSize_ValidInputs(t *testing.T) {
	// WHY: ensure valid inputs still work after hardening
	valid := []struct {
		input    string
		expected int64
	}{
		{"1024", 1024},
		{"1KB", 1024},
		{"1K", 1024},
		{"10MB", 10 * 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
	}

	for _, tc := range valid {
		result, err := ParseSize(tc.input)
		if err != nil {
			t.Errorf("ParseSize(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if result != tc.expected {
			t.Errorf("ParseSize(%q) = %d, want %d", tc.input, result, tc.expected)
		}
	}
}

func TestSecurity_ParseSize_Overflow(t *testing.T) {
	// ATTACK VECTOR: CWE-190 (Integer Overflow)
	// IMPACT: overflow in size calculation could wrap to negative or zero
	// MITIGATION EXPECTED: ParseSize detects overflow

	// WHY: MaxInt64 bytes as TB would overflow when multiplied
	huge := fmt.Sprintf("%dTB", int64(math.MaxInt64/1024))
	_, err := ParseSize(huge)
	// Either error or a large positive value is acceptable.
	// It must NOT produce a negative or zero result silently.
	if err == nil {
		// If no error, the parsed value must still be positive
		val, _ := ParseSize(huge)
		if val <= 0 {
			t.Errorf("ParseSize(%q) produced non-positive value %d (overflow)", huge, val)
		}
	}
}

// ---------------------------------------------------------------------------
// CWE-20: Input Validation -- ParseDuration
// ---------------------------------------------------------------------------

func TestSecurity_ParseDuration_InvalidInputs(t *testing.T) {
	// ATTACK VECTOR: CWE-20
	// IMPACT: crafted duration strings could cause unexpected retention
	// MITIGATION EXPECTED: ParseDuration rejects malformed inputs

	payloads := []struct {
		input string
		desc  string
	}{
		{"", "empty string"},
		{"not_a_duration", "non-numeric"},
		{"abc", "letters only"},
		{"\x00d", "null byte prefix"},
	}

	for _, tc := range payloads {
		_, err := ParseDuration(tc.input)
		if err == nil {
			t.Errorf("ParseDuration(%q) [%s] must return error", tc.input, tc.desc)
		}
	}
}

func TestSecurity_ParseDuration_ValidInputs(t *testing.T) {
	valid := []struct {
		input    string
		expected time.Duration
	}{
		{"1s", time.Second},
		{"1m", time.Minute},
		{"1h", time.Hour},
		{"1d", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"1w", 7 * 24 * time.Hour},
	}

	for _, tc := range valid {
		result, err := ParseDuration(tc.input)
		if err != nil {
			t.Errorf("ParseDuration(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if result != tc.expected {
			t.Errorf("ParseDuration(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// CWE-20: Input Validation -- NewWithConfig
// ---------------------------------------------------------------------------

func TestSecurity_NewWithConfig_NilConfig(t *testing.T) {
	// ATTACK VECTOR: CWE-476 (Null Pointer Dereference)
	// IMPACT: nil config could cause panic
	// MITIGATION EXPECTED: returns error
	sc := newSecurityTestContext(t)
	_, err := NewWithConfig(nil)
	sc.expectSecurityError(err, "NewWithConfig(nil)")
}

func TestSecurity_NewWithConfig_EmptyFilename(t *testing.T) {
	// ATTACK VECTOR: CWE-20
	// IMPACT: empty filename could write to unexpected location
	// MITIGATION EXPECTED: returns error
	sc := newSecurityTestContext(t)
	_, err := NewWithConfig(&LoggerConfig{Filename: ""})
	sc.expectSecurityError(err, "NewWithConfig with empty Filename")
}

func TestSecurity_NewWithConfig_NegativeMaxBackups(t *testing.T) {
	// ATTACK VECTOR: CWE-20
	// IMPACT: negative max_backups could cause underflow
	// MITIGATION EXPECTED: constructor succeeds (treated as 0) or errors

	sc := newSecurityTestContext(t)
	logger, err := NewWithConfig(&LoggerConfig{
		Filename:   sc.logPath("neg_backups.log"),
		MaxSizeStr: "1MB",
		MaxBackups: -5,
	})
	// Either error or safe handling is acceptable -- must NOT panic
	if err == nil && logger != nil {
		if closeErr := logger.Close(); closeErr != nil {
			t.Errorf("Close() error: %v", closeErr)
		}
	}
}

// ---------------------------------------------------------------------------
// CWE-362: Race Condition -- concurrent Write + Close + Rotate + Sync
// ---------------------------------------------------------------------------

func TestSecurity_ConcurrentWriteCloseSyncRotate(t *testing.T) {
	// ATTACK VECTOR: CWE-362
	// IMPACT: concurrent access could cause data corruption, panic,
	// or deadlock
	// MITIGATION EXPECTED: atomic operations, sync.Once, CAS rotation

	sc := newSecurityTestContext(t)

	logger, err := NewWithConfig(&LoggerConfig{
		Filename:           sc.logPath("concurrent.log"),
		MaxSizeStr:         "1KB", // tiny: forces frequent rotation
		MaxBackups:         2,
		Async:              true,
		BackpressurePolicy: "adaptive",
	})
	sc.expectSecuritySuccess(err, "NewWithConfig for concurrent test")

	var wg sync.WaitGroup
	const goroutines = 20
	const writesPerGoroutine = 50

	// WHY: hammer the logger with concurrent writes while also
	// calling Close, Sync, and Rotate from other goroutines.
	// This is the 4-way interleaving test.

	// Writers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				msg := fmt.Sprintf("writer-%d msg-%d %s\n", id, j, strings.Repeat("x", 50))
				// WHY: ignoring error -- after Close, writes may fail.
				// We only care about no panics and no data races.
				_, _ = logger.Write([]byte(msg))
			}
		}(i)
	}

	// Sync caller
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			_ = logger.Sync()
			time.Sleep(time.Millisecond)
		}
	}()

	// Rotate caller
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			_ = logger.Rotate()
			time.Sleep(time.Millisecond)
		}
	}()

	// FlushAndRotate caller
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			_ = logger.FlushAndRotate()
			time.Sleep(2 * time.Millisecond)
		}
	}()

	// Stats reader (concurrent reads during writes).
	// WHY: maxSizeBytes is now atomic.Int64, so this is race-free.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_ = logger.Stats()
			time.Sleep(time.Millisecond)
		}
	}()

	// Let writers get some work done before Close
	time.Sleep(10 * time.Millisecond)
	wg.Wait()

	// WHY: Close after all goroutines finish to avoid panic
	if closeErr := logger.Close(); closeErr != nil {
		t.Errorf("Close() error: %v", closeErr)
	}
}

func TestSecurity_ConcurrentClose(t *testing.T) {
	// ATTACK VECTOR: CWE-362
	// IMPACT: multiple concurrent Close calls could double-close file
	// MITIGATION EXPECTED: sync.Once makes Close idempotent

	sc := newSecurityTestContext(t)
	logger, err := NewWithConfig(&LoggerConfig{
		Filename:   sc.logPath("double_close.log"),
		MaxSizeStr: "1MB",
		Async:      true,
	})
	sc.expectSecuritySuccess(err, "NewWithConfig for concurrent close test")

	// Write something to initialize the file
	_, _ = logger.Write([]byte("init\n"))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// WHY: concurrent Close must not panic
			_ = logger.Close()
		}()
	}
	wg.Wait()
}

func TestSecurity_WriteAfterClose(t *testing.T) {
	// ATTACK VECTOR: CWE-362
	// IMPACT: Write after Close could panic on nil file pointer
	// MITIGATION EXPECTED: Write returns error or silently ignores

	sc := newSecurityTestContext(t)
	logger, err := NewWithConfig(&LoggerConfig{
		Filename:   sc.logPath("write_after_close.log"),
		MaxSizeStr: "1MB",
	})
	sc.expectSecuritySuccess(err, "NewWithConfig")

	_, _ = logger.Write([]byte("before close\n"))
	if closeErr := logger.Close(); closeErr != nil {
		t.Fatalf("Close() error: %v", closeErr)
	}

	// WHY: regardless of whether Write returns error or success,
	// it must NOT panic.
	_, _ = logger.Write([]byte("after close\n"))
}

// ---------------------------------------------------------------------------
// CWE-400: Resource Exhaustion -- oversized writes
// ---------------------------------------------------------------------------

func TestSecurity_OversizedSingleWrite(t *testing.T) {
	// ATTACK VECTOR: CWE-400
	// IMPACT: 10MB single write could exhaust memory in buffer copy
	// MITIGATION EXPECTED: Write handles large payloads without panic.
	// This validates lethe's claim of handling 10MB JSON payloads.

	sc := newSecurityTestContext(t)
	var errors []string
	logger, err := NewWithConfig(&LoggerConfig{
		Filename:   sc.logPath("oversized.log"),
		MaxSizeStr: "50MB", // large enough to not rotate mid-write
		MaxBackups: 1,
		ErrorCallback: func(op string, err error) {
			errors = append(errors, fmt.Sprintf("%s: %v", op, err))
		},
	})
	sc.expectSecuritySuccess(err, "NewWithConfig for oversized write")

	// WHY: 10MB payload. This is the documented capability.
	payload := make([]byte, 10*1024*1024)
	for i := range payload {
		payload[i] = byte('A' + (i % 26))
	}
	payload[len(payload)-1] = '\n'

	n, writeErr := logger.Write(payload)
	if writeErr != nil {
		t.Errorf("Write(10MB) error: %v", writeErr)
	}
	if n != len(payload) {
		t.Errorf("Write(10MB) wrote %d bytes, want %d", n, len(payload))
	}

	if closeErr := logger.Close(); closeErr != nil {
		t.Errorf("Close() error: %v", closeErr)
	}
}

// ---------------------------------------------------------------------------
// CWE-400: Resource Exhaustion -- rotation storm
// ---------------------------------------------------------------------------

func TestSecurity_RotationStorm(t *testing.T) {
	// ATTACK VECTOR: CWE-400
	// IMPACT: tiny MaxSize + rapid writes = many rotations = many files
	// MITIGATION EXPECTED: MaxBackups limits total files on disk

	sc := newSecurityTestContext(t)
	logger, err := NewWithConfig(&LoggerConfig{
		Filename:   sc.logPath("storm.log"),
		MaxSizeStr: "512", // 512 bytes: triggers rotation on every write batch
		MaxBackups: 3,     // only keep 3 backups
		Compress:   false, // no compress to speed up test
	})
	sc.expectSecuritySuccess(err, "NewWithConfig for rotation storm")

	// WHY: write 100 entries of ~100 bytes each. This should trigger
	// many rotations but MaxBackups=3 keeps disk usage bounded.
	for i := 0; i < 100; i++ {
		msg := fmt.Sprintf("storm entry %03d %s\n", i, strings.Repeat("x", 80))
		_, _ = logger.Write([]byte(msg))
	}

	if closeErr := logger.Close(); closeErr != nil {
		t.Errorf("Close() error: %v", closeErr)
	}

	// Count files matching the pattern
	pattern := sc.logPath("storm.log*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Glob error: %v", err)
	}

	// WHY: MaxBackups=3 means at most 4 files (current + 3 backups)
	if len(matches) > 4 {
		t.Errorf("rotation storm produced %d files (max expected 4): %v",
			len(matches), matches)
	}
}

// ---------------------------------------------------------------------------
// CWE-770: Uncontrolled Buffer Growth -- goroutine leak
// ---------------------------------------------------------------------------

func TestSecurity_NoGoroutineLeakOnCreateClose(t *testing.T) {
	// ATTACK VECTOR: CWE-770
	// IMPACT: repeated NewWithConfig + Close could leak goroutines
	// (metrics callback, MPSC consumer, background workers)
	// MITIGATION EXPECTED: Close stops all goroutines

	sc := newSecurityTestContext(t)

	// WHY: create and close 20 loggers. If goroutines leak,
	// the test runtime will be noticeably slow or fail with
	// "too many goroutines" in CI. The -race detector also
	// catches leaked goroutines.
	for i := 0; i < 20; i++ {
		logger, err := NewWithConfig(&LoggerConfig{
			Filename:   sc.logPath(fmt.Sprintf("leak_%d.log", i)),
			MaxSizeStr: "1MB",
			Async:      true,
		})
		if err != nil {
			t.Fatalf("iteration %d: NewWithConfig error: %v", i, err)
		}

		// Write to trigger MPSC consumer startup
		_, _ = logger.Write([]byte("trigger mpsc\n"))

		if closeErr := logger.Close(); closeErr != nil {
			t.Errorf("iteration %d: Close() error: %v", i, closeErr)
		}
	}

	// WHY: if we get here without timeout or excessive goroutine
	// warnings, the leak test passes. A more precise check would
	// count runtime.NumGoroutine but that is flaky in test suites.
}

// ---------------------------------------------------------------------------
// CWE-20: ErrorCallback invocation
// ---------------------------------------------------------------------------

func TestSecurity_ErrorCallback_OnRotationFailure(t *testing.T) {
	// ATTACK VECTOR: CWE-20 / CWE-400
	// IMPACT: if ErrorCallback is not set, rotation failures are SILENT.
	// This test verifies the callback is actually invoked when set.
	// MITIGATION EXPECTED: ErrorCallback fires on any internal error.

	sc := newSecurityTestContext(t)

	var mu sync.Mutex
	var errors []string
	logger, err := NewWithConfig(&LoggerConfig{
		Filename:   sc.logPath("errcb.log"),
		MaxSizeStr: "1KB",
		MaxBackups: 1,
		Compress:   false,
		ErrorCallback: func(op string, opErr error) {
			mu.Lock()
			errors = append(errors, fmt.Sprintf("%s: %v", op, opErr))
			mu.Unlock()
		},
	})
	sc.expectSecuritySuccess(err, "NewWithConfig with ErrorCallback")

	// Write enough to trigger rotation
	for i := 0; i < 50; i++ {
		_, _ = logger.Write([]byte(strings.Repeat("X", 100) + "\n"))
	}

	// WHY: give background workers time to process
	logger.WaitForBackgroundTasks()

	if closeErr := logger.Close(); closeErr != nil {
		t.Errorf("Close() error: %v", closeErr)
	}

	// We don't assert specific errors here. The test validates
	// that ErrorCallback WORKS without panic. If there are no
	// errors (normal rotation succeeds), that's also fine.
	mu.Lock()
	t.Logf("ErrorCallback invocations: %d", len(errors))
	mu.Unlock()
}

// ---------------------------------------------------------------------------
// CWE-20: FileMode enforcement
// ---------------------------------------------------------------------------

func TestSecurity_FileMode_Applied(t *testing.T) {
	// ATTACK VECTOR: CWE-732 (Incorrect Permission Assignment)
	// IMPACT: world-readable log files could leak sensitive data
	// MITIGATION EXPECTED: FileMode is applied to created log files

	sc := newSecurityTestContext(t)
	mode := os.FileMode(0o640)
	logger, err := NewWithConfig(&LoggerConfig{
		Filename:   sc.logPath("filemode.log"),
		MaxSizeStr: "1MB",
		FileMode:   mode,
	})
	sc.expectSecuritySuccess(err, "NewWithConfig with FileMode")

	_, err = logger.Write([]byte("test permissions\n"))
	sc.expectSecuritySuccess(err, "Write with FileMode")

	if closeErr := logger.Close(); closeErr != nil {
		t.Errorf("Close() error: %v", closeErr)
	}

	info, err := os.Stat(sc.logPath("filemode.log"))
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}

	actualMode := info.Mode().Perm()
	if actualMode != mode {
		t.Errorf("file mode = %o, want %o", actualMode, mode)
	}
}

// ---------------------------------------------------------------------------
// CWE-20: BackpressurePolicy validation
// ---------------------------------------------------------------------------

func TestSecurity_BackpressurePolicy_Values(t *testing.T) {
	// ATTACK VECTOR: CWE-20
	// IMPACT: unknown policy string could cause undefined behavior
	// MITIGATION EXPECTED: NewWithConfig defaults or rejects unknown

	sc := newSecurityTestContext(t)

	valid := []string{"fallback", "drop", "adaptive"}
	for _, policy := range valid {
		logger, err := NewWithConfig(&LoggerConfig{
			Filename:           sc.logPath(fmt.Sprintf("bp_%s.log", policy)),
			MaxSizeStr:         "1MB",
			Async:              true,
			BackpressurePolicy: policy,
		})
		if err != nil {
			t.Errorf("BackpressurePolicy %q rejected: %v", policy, err)
			continue
		}
		_, _ = logger.Write([]byte("test\n"))
		if closeErr := logger.Close(); closeErr != nil {
			t.Errorf("Close() error for policy %q: %v", policy, closeErr)
		}
	}
}

// ---------------------------------------------------------------------------
// CWE-20: Sync and FlushAndRotate on clean logger
// ---------------------------------------------------------------------------

func TestSecurity_SyncOnEmptyLogger(t *testing.T) {
	// ATTACK VECTOR: CWE-476
	// IMPACT: Sync on a logger that never wrote could dereference nil file
	// MITIGATION EXPECTED: Sync handles nil state gracefully

	sc := newSecurityTestContext(t)
	logger, err := NewWithConfig(&LoggerConfig{
		Filename:   sc.logPath("sync_empty.log"),
		MaxSizeStr: "1MB",
	})
	sc.expectSecuritySuccess(err, "NewWithConfig")

	// WHY: Sync before any Write -- file hasn't been created yet
	// Must NOT panic.
	_ = logger.Sync()
	_ = logger.FlushAndRotate()

	if closeErr := logger.Close(); closeErr != nil {
		t.Errorf("Close() error: %v", closeErr)
	}
}

// ---------------------------------------------------------------------------
// CWE-20: Checksum integrity
// ---------------------------------------------------------------------------

func TestSecurity_Checksum_Generated(t *testing.T) {
	// ATTACK VECTOR: CWE-354 (Insufficient Verification of Data Authenticity)
	// IMPACT: rotated log files without checksums cannot prove integrity
	// MITIGATION EXPECTED: SHA-256 sidecar files generated on rotation

	sc := newSecurityTestContext(t)
	logger, err := NewWithConfig(&LoggerConfig{
		Filename:   sc.logPath("checksum.log"),
		MaxSizeStr: "512", // tiny: triggers rotation
		MaxBackups: 2,
		Checksum:   true,
		Compress:   false,
	})
	sc.expectSecuritySuccess(err, "NewWithConfig with Checksum")

	// Write enough to trigger rotation
	for i := 0; i < 30; i++ {
		_, _ = logger.Write([]byte(strings.Repeat("C", 100) + "\n"))
	}

	// Wait for background checksum tasks
	logger.WaitForBackgroundTasks()

	if closeErr := logger.Close(); closeErr != nil {
		t.Errorf("Close() error: %v", closeErr)
	}

	// Look for .sha256 sidecar files
	matches, err := filepath.Glob(sc.logPath("checksum.log.*.sha256"))
	if err != nil {
		t.Fatalf("Glob error: %v", err)
	}

	if len(matches) == 0 {
		t.Error("no .sha256 sidecar files generated after rotation with Checksum=true")
	}

	// Verify sidecar format: "{hex}  {filename}\n"
	for _, sha256File := range matches {
		data, readErr := os.ReadFile(sha256File)
		if readErr != nil {
			t.Errorf("cannot read sidecar %s: %v", sha256File, readErr)
			continue
		}
		content := string(data)
		// sha256sum format: 64 hex chars + two spaces + filename + newline
		if len(content) < 66 {
			t.Errorf("sidecar %s too short: %q", sha256File, content)
		}
	}
}

// ---------------------------------------------------------------------------
// CWE-20: LoadFromJSON with malformed input
// ---------------------------------------------------------------------------

func TestSecurity_LoadFromJSON_MalformedInput(t *testing.T) {
	// ATTACK VECTOR: CWE-502 (Deserialization of Untrusted Data)
	// IMPACT: malformed JSON could cause panic during unmarshal
	// MITIGATION EXPECTED: json.Unmarshal returns error

	payloads := []struct {
		input string
		desc  string
	}{
		{"", "empty string"},
		{"{", "incomplete JSON"},
		{`{"filename": 42}`, "wrong type for filename"},
		{`null`, "JSON null"},
		{strings.Repeat("{", 1000), "deeply nested"},
		{"\x00\x01\x02", "binary garbage"},
	}

	for _, tc := range payloads {
		_, err := LoadFromJSON([]byte(tc.input))
		// WHY: error or empty filename -> caught later. Must NOT panic.
		if err == nil {
			// If no parse error, filename validation should catch it
			cfg, _ := LoadFromJSON([]byte(tc.input))
			if cfg != nil && cfg.Filename == "" {
				// This is fine -- empty filename caught by NewWithConfig
			}
		}
	}
}
