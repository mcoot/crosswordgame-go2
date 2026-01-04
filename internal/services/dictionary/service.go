package dictionary

import (
	"bufio"
	"context"
	"os"
	"strings"
	"sync"

	"github.com/mcoot/crosswordgame-go2/internal/model"
	"github.com/mcoot/crosswordgame-go2/internal/storage"
)

// Service provides dictionary/word validation functionality
type Service struct {
	storage storage.Storage

	mu     sync.RWMutex
	words  map[string]struct{}
	loaded bool
}

// New creates a new DictionaryService
func New(storage storage.Storage) *Service {
	return &Service{
		storage: storage,
		words:   make(map[string]struct{}),
	}
}

// LoadFromStorage loads dictionary words from storage
func (s *Service) LoadFromStorage(ctx context.Context) error {
	words, err := s.storage.GetDictionaryWords(ctx)
	if err != nil {
		return err
	}
	return s.loadWords(words)
}

// LoadFromFile loads dictionary words from a file (one word per line)
func (s *Service) LoadFromFile(ctx context.Context, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			words = append(words, word)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Save to storage for future use
	if err := s.storage.SaveDictionaryWords(ctx, words); err != nil {
		return err
	}

	return s.loadWords(words)
}

// LoadWords directly loads a slice of words (useful for testing)
func (s *Service) LoadWords(words []string) error {
	return s.loadWords(words)
}

func (s *Service) loadWords(words []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.words = make(map[string]struct{}, len(words))
	for _, word := range words {
		// Store lowercase for case-insensitive matching
		s.words[strings.ToLower(word)] = struct{}{}
	}
	s.loaded = true
	return nil
}

// IsValidWord checks if a word exists in the dictionary
// Words must be at least 2 characters
func (s *Service) IsValidWord(word string) bool {
	if len(word) < 2 {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.loaded {
		return false
	}

	_, ok := s.words[strings.ToLower(word)]
	return ok
}

// IsLoaded returns whether the dictionary has been loaded
func (s *Service) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}

// WordCount returns the number of words in the dictionary
func (s *Service) WordCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.words)
}

// FindAllValidWords finds all valid words in a line of letters
// Returns all valid substrings of length >= 2
func (s *Service) FindAllValidWords(letters []rune) []ValidWord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.loaded {
		return nil
	}

	var results []ValidWord
	n := len(letters)

	for start := 0; start < n; start++ {
		for end := start + 2; end <= n; end++ {
			word := string(letters[start:end])
			if _, ok := s.words[strings.ToLower(word)]; ok {
				results = append(results, ValidWord{
					Word:  word,
					Start: start,
					End:   end,
				})
			}
		}
	}

	return results
}

// ValidWord represents a valid word found in a sequence of letters
type ValidWord struct {
	Word  string
	Start int // Inclusive
	End   int // Exclusive
}

// Interface check
type ServiceInterface interface {
	IsValidWord(word string) bool
	IsLoaded() bool
	WordCount() int
	FindAllValidWords(letters []rune) []ValidWord
	LoadFromStorage(ctx context.Context) error
	LoadFromFile(ctx context.Context, path string) error
	LoadWords(words []string) error
}

var _ ServiceInterface = (*Service)(nil)

// ErrDictionaryNotLoaded is returned when operations are attempted before loading
var ErrDictionaryNotLoaded = model.ErrDictionaryNotLoaded
