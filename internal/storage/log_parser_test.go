package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testLogLine() string {
	timestamp := time.Now().Format("02/Jan/2006:15:04:05 -0700")
	return `127.0.0.1 - - [` + timestamp + `] "GET /hello%20world HTTP/1.1" 200 123 "-" "NixVisTest/1.0"`
}

func TestParseNginxLogLine(t *testing.T) {
	parser := &LogParser{}
	record, err := parser.parseNginxLogLine(testLogLine())
	if err != nil {
		t.Fatalf("parseNginxLogLine returned an error: %v", err)
	}
	if record.Url != "/hello world" {
		t.Fatalf("unexpected decoded URL: %q", record.Url)
	}
	if record.IP != "127.0.0.1" {
		t.Fatalf("unexpected IP: %q", record.IP)
	}
}

func TestUpdateStateRoundTrip(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "nginx_scan_state.json")
	parser := &LogParser{
		statePath: statePath,
		states: map[string]LogScanState{
			"site": {Files: map[string]FileState{"access.log": {LastOffset: 12, LastSize: 18}}},
		},
	}
	parser.updateState()

	reloaded := &LogParser{statePath: statePath}
	reloaded.loadState()
	state := reloaded.states["site"].Files["access.log"]
	if state.LastOffset != 12 || state.LastSize != 18 {
		t.Fatalf("unexpected persisted state: %+v", state)
	}
}

func TestParseLogLinesReportsDatabaseFailure(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	db.Close()

	file, err := os.CreateTemp(t.TempDir(), "access.log")
	if err != nil {
		t.Fatalf("create log file: %v", err)
	}
	defer file.Close()
	if _, err := file.WriteString(testLogLine() + "\n"); err != nil {
		t.Fatalf("write log file: %v", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("seek log file: %v", err)
	}

	parser := &LogParser{repo: &Repository{db: db}}
	if entries := parser.parseLogLines(file, "site", &ParserResult{}); entries != -1 {
		t.Fatalf("expected database failure marker, got %d", entries)
	}
}
