package legacy

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func testDocuments() map[Product][]byte {
	return map[Product][]byte{
		Stats:       []byte("{\"stats\":true}\n"),
		GlobalStats: []byte("{\"global\":true}\n"),
	}
}

func TestPublish(t *testing.T) {
	dir := t.TempDir()
	documents := map[Product][]byte{GlobalStats: testDocuments()[GlobalStats]}
	if err := Publish(dir, documents, false); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "js", "global_stats.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Publish(dir, documents, false); err == nil {
		t.Fatal("existing file was overwritten without --overwrite")
	}
	after, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(after, before) {
		t.Fatal("refused publication changed the existing file")
	}
}
