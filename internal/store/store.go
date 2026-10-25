package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mobile-agent/internal/types"
)

type Store struct {
	dir string
}

func New(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *Store) Save(sess *types.Session) error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return err
	}
	sess.Updated = time.Now()
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(sess.ID), data, 0600)
}

func (s *Store) Load(id string) (*types.Session, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	var sess types.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) Delete(id string) error {
	return os.Remove(s.path(id))
}

func (s *Store) List() ([]*types.Session, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var sessions []*types.Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var sess types.Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}
		sessions = append(sessions, &sess)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Updated.After(sessions[j].Updated)
	})
	return sessions, nil
}

func NewID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString(b)
	}
	return hex.EncodeToString(b)
}

func NewSession(id, provider, model, stack, cwd string) *types.Session {
	now := time.Now()
	return &types.Session{
		ID:       id,
		Provider: provider,
		Model:    model,
		Stack:    stack,
		CWD:      cwd,
		Created:  now,
		Updated:  now,
	}
}
