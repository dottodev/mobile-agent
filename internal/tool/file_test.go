package tool

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFileTool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	f := FileTool{}

	res, err := f.Call(context.Background(), Input{
		ID:        "1",
		Name:      "file",
		Arguments: `{"op":"write","path":"` + path + `","content":"hello"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Error {
		t.Fatalf("write failed: %s", res.Output)
	}

	res, err = f.Call(context.Background(), Input{
		ID:        "2",
		Name:      "file",
		Arguments: `{"op":"read","path":"` + path + `"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output != "hello" {
		t.Fatalf("unexpected content %q", res.Output)
	}

	res, err = f.Call(context.Background(), Input{
		ID:        "3",
		Name:      "file",
		Arguments: `{"op":"edit","path":"` + path + `","old":"hello","new":"world"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Error {
		t.Fatalf("edit failed: %s", res.Output)
	}

	res, err = f.Call(context.Background(), Input{
		ID:        "4",
		Name:      "file",
		Arguments: `{"op":"read","path":"` + path + `"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output != "world" {
		t.Fatalf("unexpected content %q", res.Output)
	}
}
