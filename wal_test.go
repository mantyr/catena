package catena

import (
	"reflect"
	"testing"
)

func TestWAL(t *testing.T) {
	// Open a new test WAL file. We truncate any existing file.
	w, err := newFileWAL("/tmp/test.wal")
	if err != nil {
		t.Fatal(err)
	}

	if w.lastReadOffset != 0 {
		t.Error("expected lastReadOffset to be 0, got %d", w.lastReadOffset)
	}

	entry := walEntry{
		operation: operationInsert,
		rows: []walRow{
			walRow{
				Source:    "hostA",
				Metric:    "metric.1",
				Timestamp: 123,
				Value:     0.234,
			},
		},
	}

	n, err := w.append(entry)
	if err != nil {
		t.Fatal(err)
	}

	if n == 0 {
		t.Errorf("expected to get non-zero bytes written, got %d", n)
	}

	err = w.close()
	if err != nil {
		t.Fatal(err)
	}

	w, err = openFileWAL("/tmp/test.wal")
	if err != nil {
		t.Fatal(err)
	}

	readEntry, err := w.readEntry()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(readEntry, entry) {
		t.Errorf("expected walEntry %#v, got %#v", entry, readEntry)
	}

	err = w.close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestWALWithBadEntry(t *testing.T) {
	w, err := newFileWAL("/tmp/test.wal")
	if err != nil {
		t.Fatal(err)
	}

	if w.lastReadOffset != 0 {
		t.Error("expected lastReadOffset to be 0, got %d", w.lastReadOffset)
	}

	entry := walEntry{
		operation: operationInsert,
		rows: []walRow{
			walRow{
				Source:    "hostA",
				Metric:    "metric.1",
				Timestamp: 123,
				Value:     0.234,
			},
			walRow{
				Source:    "hostB",
				Metric:    "metric.1",
				Timestamp: 123,
				Value:     0.234,
			},
			walRow{
				Source:    "hostA",
				Metric:    "metric.2",
				Timestamp: 123,
				Value:     0.234,
			},
		},
	}

	n, err := w.append(entry)
	if err != nil {
		t.Fatal(err)
	}

	if n == 0 {
		t.Errorf("expected to get non-zero bytes written, got %d", n)
	}

	offset, err := w.f.Seek(0, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Now append some garbage, but add a correct magic sequence.
	// Note: the magic sequence is in little-endian.
	_, err = w.f.WriteString("\x93\x19\x14\x11 garbage!")
	if err != nil {
		t.Fatal(err)
	}
	err = w.f.Sync()
	if err != nil {
		t.Fatal(err)
	}

	err = w.close()
	if err != nil {
		t.Fatal(err)
	}

	w, err = openFileWAL("/tmp/test.wal")
	if err != nil {
		t.Fatal(err)
	}

	readEntry, err := w.readEntry()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(readEntry, entry) {
		t.Errorf("expected walEntry %#v, got %#v", entry, readEntry)
	}

	// Now try to read the garbage.
	readEntry, err = w.readEntry()
	if err == nil {
		t.Fatalf("expected to get an error after reading garbage, but got %#v", readEntry)
	}

	// Check last good offset
	if w.lastReadOffset != offset {
		t.Errorf("expected lastReadOffset to be %d, got %d", offset, w.lastReadOffset)
	}

	// Test truncate
	err = w.truncate()
	if err != nil {
		t.Fatal(err)
	}

	// Close and read again.
	err = w.close()
	if err != nil {
		t.Fatal(err)
	}

	w, err = openFileWAL("/tmp/test.wal")
	if err != nil {
		t.Fatal(err)
	}

	readEntry, err = w.readEntry()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(readEntry, entry) {
		t.Errorf("expected walEntry %#v, got %#v", entry, readEntry)
	}

	// Now try to read past the end.
	readEntry, err = w.readEntry()
	if err == nil {
		t.Fatalf("expected to get an error attempting to read past the end, but got %#v", readEntry)
	}
}
