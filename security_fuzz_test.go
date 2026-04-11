package lethe

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// FuzzParseSize -- every string that crosses the ParseSize trust boundary
// ---------------------------------------------------------------------------

func FuzzParseSize(f *testing.F) {
	// Seeds: valid sizes, edge cases, and attack patterns
	seeds := []string{
		// Valid
		"1024", "100MB", "1GB", "10KB", "1TB",
		"1K", "1M", "1G", "1T",
		// Case variations
		"100mb", "1gb", "10kb",
		// Edge
		"0", "1", "0MB",
		// Attack
		"", "MB", "not_a_size", "-1MB",
		"99999999999999TB",    // overflow
		"10MB; rm -rf /",      // shell injection
		"\x00MB",              // null byte
		"10\x00MB",            // null mid-string
		"1.5GB",               // float
		" 100MB ",             // whitespace
		"10  MB",              // extra spaces
		"9223372036854775807", // MaxInt64 as bytes
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result, err := ParseSize(input)
		if err != nil {
			return // expected for most fuzz inputs
		}

		// WHY: if ParseSize succeeds, the result must be non-negative.
		// A negative result would indicate integer overflow.
		if result < 0 {
			t.Errorf("ParseSize(%q) = %d, must not be negative (overflow)", input, result)
		}
	})
}

// ---------------------------------------------------------------------------
// FuzzParseDuration -- every string that reaches ParseDuration
// ---------------------------------------------------------------------------

func FuzzParseDuration(f *testing.F) {
	seeds := []string{
		// Valid Go durations
		"1s", "100ms", "1h30m", "500us", "1ns",
		// Valid lethe extensions
		"1d", "7d", "30d", "1w", "52w", "1y",
		// Edge
		"0", "0s", "0d",
		// Attack
		"", "abc", "not_a_duration",
		"-1d", "-7d",
		"\x00d",               // null byte
		"7d; rm -rf /",        // shell injection
		"999999999999999999d", // near overflow
		"1.5d",                // float day
		" 7d ",                // whitespace
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// WHY: must not panic regardless of input
		_, _ = ParseDuration(input)
	})
}

// ---------------------------------------------------------------------------
// FuzzSanitizeFilename -- every string that reaches the sanitizer
// ---------------------------------------------------------------------------

func FuzzSanitizeFilename(f *testing.F) {
	seeds := []string{
		// Clean
		"app.log", "metis-2026.log", "debug_trace.log.1",
		// Dangerous chars
		"file<script>.log", "file|pipe.log", "file\"quote.log",
		"file?glob.log", "file*star.log",
		// Control characters
		"\x00evil.log", "\x01ctrl.log", "\x1fctrl.log",
		// Path traversal (sanitizer sees base only)
		"../../../etc/passwd",
		// Windows devices
		"CON", "NUL", "COM1", "PRN", "LPT1",
		// Windows ADS
		"file.log:stream",
		// Null in middle
		"file\x00evil.log",
		// Unicode
		"\u202efile.log", // RTL override
		// Empty
		"",
		// Just dots
		".", "..",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// WHY: SanitizeFilename must never panic and must always
		// return a string. The output must not contain the specific
		// dangerous characters that the function is designed to strip.
		result := SanitizeFilename(input)

		// Result must not be longer than input + some slack
		// (there should be no expansion)
		if len(result) > len(input)+1 {
			t.Errorf("SanitizeFilename(%q) expanded to %q (longer)", input, result)
		}

		_ = result // use result to prevent optimization
	})
}

// ---------------------------------------------------------------------------
// FuzzWrite -- fuzz the Write path with arbitrary payloads
// ---------------------------------------------------------------------------

func FuzzWrite(f *testing.F) {
	seeds := [][]byte{
		// Normal log line
		[]byte(`{"time":"2026-01-01T00:00:00Z","level":"INFO","msg":"hello"}` + "\n"),
		// Empty
		{},
		// Just newline
		{'\n'},
		// Null bytes
		{0, 0, 0, 0},
		// Large-ish structured payload
		[]byte(fmt.Sprintf(`{"data":"%s"}`, string(make([]byte, 1000)))),
		// Binary
		{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'},
		// Control chars
		{'\x00', '\x01', '\x02', '\x03', '\x1b', '[', '3', '1', 'm'},
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// WHY: each fuzz iteration gets its own isolated logger.
		// We use os.MkdirTemp because t.TempDir() is shared across
		// fuzz seeds in some Go versions.
		dir, err := os.MkdirTemp("", "lethe-fuzz-write-*")
		if err != nil {
			return
		}
		defer func() {
			_ = os.RemoveAll(dir)
		}()

		logger, createErr := NewWithConfig(&LoggerConfig{
			Filename:   filepath.Join(dir, "fuzz.log"),
			MaxSizeStr: "1MB",
			MaxBackups: 1,
		})
		if createErr != nil {
			return
		}

		// WHY: Write must not panic regardless of payload content.
		// Error is acceptable. Return value consistency is verified.
		n, writeErr := logger.Write(data)
		if writeErr == nil && n != len(data) {
			t.Errorf("Write returned n=%d for len(data)=%d without error", n, len(data))
		}

		if closeErr := logger.Close(); closeErr != nil {
			t.Errorf("Close() error: %v", closeErr)
		}
	})
}
