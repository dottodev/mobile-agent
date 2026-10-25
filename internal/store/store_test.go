package store

import (
	"testing"
)

func TestStoreSaveLoadList(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	sess := NewSession(NewID(), "openai", "model", "build", "/tmp")
	sess.Title = "test session"

	if err := s.Save(sess); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.Load(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "test session" {
		t.Fatalf("unexpected title %q", loaded.Title)
	}

	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}

	if err := s.Delete(sess.ID); err != nil {
		t.Fatal(err)
	}
	list, err = s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(list))
	}
}
