package main

import (
	"encoding/base32"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Session states.
const stateOpen = "Open"

// Session is one Secret Santa round as persisted.
type Session struct {
	Hash         string
	State        string
	Lang         string // the Organizer's language, used for invitation emails
	Participants []Participant
}

// Participant is one person taking part in a Session.
type Participant struct {
	Name      string
	Email     string
	Hash      string // Personal link hash
	Organizer bool
}

var (
	// ErrSessionExists is returned by Store.Create for an already used hash.
	ErrSessionExists = errors.New("session already exists")

	hashPattern  = regexp.MustCompile(`^[a-z2-7]{26}$`)
	hashEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)
)

// validHash reports whether s has the shape of a generated hash. Checking it
// before any lookup also keeps hashes safe to use as file names.
func validHash(s string) bool { return hashPattern.MatchString(s) }

// newHash returns a secret hash: 128 random bits as 26 lowercase base32 characters.
func newHash(random io.Reader) (string, error) {
	var b [16]byte
	if _, err := io.ReadFull(random, b[:]); err != nil {
		return "", err
	}
	return strings.ToLower(hashEncoding.EncodeToString(b[:])), nil
}

// Store persists Sessions as one JSON file each in a directory (ADR-0001).
// It is safe for concurrent use; callers never see files or the Personal
// link index.
type Store struct {
	dir string
	mu  sync.Mutex
	// sessionOf maps a Personal link hash to the hash of its Session.
	sessionOf map[string]string
}

// NewStore opens the directory (creating it if needed) and indexes the
// Personal links of all stored Sessions.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, sessionOf: map[string]string{}}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		sess, err := readSession(f)
		if err != nil {
			return nil, err
		}
		s.index(sess)
	}
	return s, nil
}

// Create stores a new Session, failing with ErrSessionExists if its hash is taken.
func (s *Store) Create(sess Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.path(sess.Hash)
	if _, err := os.Stat(path); err == nil {
		return ErrSessionExists
	}
	if err := writeAtomically(path, sess); err != nil {
		return err
	}
	s.index(sess)
	return nil
}

// Load returns the Session with the given hash; ok is false if there is none.
func (s *Store) Load(hash string) (sess Session, ok bool, err error) {
	if !validHash(hash) {
		return Session{}, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, err = readSession(s.path(hash))
	if errors.Is(err, os.ErrNotExist) {
		return Session{}, false, nil
	}
	return sess, err == nil, err
}

// LoadByPersonalLink returns the Session containing the Participant with the
// given Personal link hash; ok is false if there is none.
func (s *Store) LoadByPersonalLink(hash string) (Session, bool, error) {
	s.mu.Lock()
	sessionHash, found := s.sessionOf[hash]
	s.mu.Unlock()
	if !found {
		return Session{}, false, nil
	}
	return s.Load(sessionHash)
}

func (s *Store) path(hash string) string { return filepath.Join(s.dir, hash+".json") }

func (s *Store) index(sess Session) {
	for _, p := range sess.Participants {
		s.sessionOf[p.Hash] = sess.Hash
	}
}

func readSession(path string) (Session, error) {
	var sess Session
	data, err := os.ReadFile(path)
	if err != nil {
		return sess, err
	}
	err = json.Unmarshal(data, &sess)
	return sess, err
}

// writeAtomically writes sess to a temp file and renames it over path, so
// readers never see a partial file.
func writeAtomically(path string, sess Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
