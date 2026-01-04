package dictionary

import (
	"context"
	"testing"

	"github.com/mcoot/crosswordgame-go2/internal/storage/memory"
	"github.com/stretchr/testify/suite"
)

type ServiceSuite struct {
	suite.Suite
	storage *memory.Storage
	service *Service
	ctx     context.Context
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) SetupTest() {
	s.storage = memory.New()
	s.service = New(s.storage)
	s.ctx = context.Background()
}

func (s *ServiceSuite) TestIsNotLoadedByDefault() {
	s.False(s.service.IsLoaded())
	s.Equal(0, s.service.WordCount())
}

func (s *ServiceSuite) TestLoadWords() {
	words := []string{"apple", "banana", "cherry"}
	err := s.service.LoadWords(words)
	s.Require().NoError(err)

	s.True(s.service.IsLoaded())
	s.Equal(3, s.service.WordCount())
}

func (s *ServiceSuite) TestIsValidWordAfterLoading() {
	words := []string{"apple", "banana", "cherry"}
	_ = s.service.LoadWords(words)

	s.True(s.service.IsValidWord("apple"))
	s.True(s.service.IsValidWord("banana"))
	s.True(s.service.IsValidWord("cherry"))
	s.False(s.service.IsValidWord("grape"))
}

func (s *ServiceSuite) TestIsValidWordCaseInsensitive() {
	words := []string{"Apple", "BANANA"}
	_ = s.service.LoadWords(words)

	s.True(s.service.IsValidWord("apple"))
	s.True(s.service.IsValidWord("APPLE"))
	s.True(s.service.IsValidWord("Apple"))
	s.True(s.service.IsValidWord("banana"))
	s.True(s.service.IsValidWord("BANANA"))
}

func (s *ServiceSuite) TestIsValidWordRequiresMinLength() {
	words := []string{"a", "ab", "abc"}
	_ = s.service.LoadWords(words)

	s.False(s.service.IsValidWord("a"))  // Too short (stored but rejected)
	s.True(s.service.IsValidWord("ab"))  // Minimum length
	s.True(s.service.IsValidWord("abc")) // Valid
}

func (s *ServiceSuite) TestIsValidWordWhenNotLoaded() {
	s.False(s.service.IsValidWord("apple"))
}

func (s *ServiceSuite) TestLoadFromStorage() {
	// Pre-populate storage with words
	words := []string{"test", "word", "example"}
	err := s.storage.SaveDictionaryWords(s.ctx, words)
	s.Require().NoError(err)

	err = s.service.LoadFromStorage(s.ctx)
	s.Require().NoError(err)

	s.True(s.service.IsLoaded())
	s.Equal(3, s.service.WordCount())
	s.True(s.service.IsValidWord("test"))
}

func (s *ServiceSuite) TestLoadFromStorageWhenEmpty() {
	err := s.service.LoadFromStorage(s.ctx)
	s.ErrorIs(err, ErrDictionaryNotLoaded)
}

func (s *ServiceSuite) TestFindAllValidWords() {
	words := []string{"at", "ate", "eat", "tea", "eating"}
	_ = s.service.LoadWords(words)

	// Test with "EAT" - should find "EA" if it were valid, "AT", "EAT"
	letters := []rune{'E', 'A', 'T'}
	results := s.service.FindAllValidWords(letters)

	// Should find: AT (positions 1-3), EAT (positions 0-3)
	s.Require().Len(results, 2)

	foundAT := false
	foundEAT := false
	for _, r := range results {
		if r.Word == "AT" && r.Start == 1 && r.End == 3 {
			foundAT = true
		}
		if r.Word == "EAT" && r.Start == 0 && r.End == 3 {
			foundEAT = true
		}
	}
	s.True(foundAT, "should find AT")
	s.True(foundEAT, "should find EAT")
}

func (s *ServiceSuite) TestFindAllValidWordsWhenNotLoaded() {
	letters := []rune{'T', 'E', 'S', 'T'}
	results := s.service.FindAllValidWords(letters)
	s.Nil(results)
}

func (s *ServiceSuite) TestFindAllValidWordsEmptyInput() {
	_ = s.service.LoadWords([]string{"test"})
	results := s.service.FindAllValidWords([]rune{})
	s.Empty(results)
}

func (s *ServiceSuite) TestFindAllValidWordsSingleLetter() {
	_ = s.service.LoadWords([]string{"a", "test"})
	results := s.service.FindAllValidWords([]rune{'A'})
	s.Empty(results) // Single letter is too short
}
