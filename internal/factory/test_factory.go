package factory

import (
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/dependencies/mocks"
	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
	"github.com/mcoot/crosswordgame-go2/internal/storage/memory"
)

// TestApp extends App with test-specific helpers
type TestApp struct {
	*App

	// Mocks for test control
	MockClock  *mocks.MockClock
	MockRandom *mocks.MockRandom
}

// NewTestApp creates an App configured for testing with mocked dependencies
func NewTestApp() *TestApp {
	store := memory.New()
	mockClock := mocks.NewMockClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	mockRandom := mocks.NewMockRandom()

	app := newWithDependencies(store, mockClock, mockRandom, auth.DefaultConfig())

	return &TestApp{
		App:        app,
		MockClock:  mockClock,
		MockRandom: mockRandom,
	}
}

// LoadTestDictionary loads a small dictionary for testing
func (t *TestApp) LoadTestDictionary() error {
	words := []string{
		// 2-letter words
		"at", "be", "do", "go", "he", "if", "in", "is", "it", "me",
		"my", "no", "of", "on", "or", "so", "to", "up", "us", "we",
		// 3-letter words
		"ace", "act", "add", "age", "ago", "aid", "aim", "air", "all", "and",
		"ant", "any", "ape", "arc", "are", "ark", "arm", "art", "ash", "ask",
		"ate", "bad", "bag", "ban", "bar", "bat", "bed", "bee", "bet", "big",
		"bit", "box", "boy", "bug", "bus", "but", "buy", "cab", "can", "cap",
		"car", "cat", "cop", "cow", "cry", "cup", "cut", "day", "did", "die",
		"dig", "dip", "dog", "dot", "dry", "due", "dug", "ear", "eat", "egg",
		"end", "era", "eve", "eye", "fan", "far", "fat", "few", "fig", "fin",
		"fit", "fix", "fly", "fog", "for", "fox", "fun", "fur", "gap", "gas",
		"get", "god", "got", "gun", "gut", "guy", "had", "ham", "has", "hat",
		"hay", "hen", "her", "hid", "him", "hip", "his", "hit", "hog", "hop",
		"hot", "how", "hug", "ice", "ill", "ink", "inn", "ion", "its", "jam",
		"jar", "jaw", "jet", "job", "jog", "joy", "jug", "key", "kid", "kit",
		"lap", "law", "lay", "led", "leg", "let", "lid", "lie", "lip", "lit",
		"log", "lot", "low", "mad", "man", "map", "mat", "may", "men", "met",
		"mid", "mix", "mob", "mom", "mop", "mud", "mug", "nap", "net", "new",
		"nod", "nor", "not", "now", "nut", "oak", "odd", "off", "oil", "old",
		"one", "opt", "orb", "ore", "our", "out", "owe", "owl", "own", "pad",
		"pan", "pat", "paw", "pay", "pea", "pen", "per", "pet", "pie", "pig",
		"pin", "pit", "pod", "pop", "pot", "put", "ran", "rat", "raw", "ray",
		"red", "rib", "rid", "rig", "rim", "rip", "rob", "rod", "rot", "row",
		"rub", "rug", "run", "sad", "sat", "saw", "say", "sea", "set", "she",
		"shy", "sin", "sip", "sir", "sis", "sit", "six", "ski", "sky", "sly",
		"sob", "sod", "son", "sop", "sow", "spa", "spy", "sub", "sum", "sun",
		"tab", "tag", "tan", "tap", "tar", "tax", "tea", "ten", "the", "thy",
		"tie", "tin", "tip", "toe", "ton", "too", "top", "tow", "toy", "try",
		"tub", "tug", "two", "urn", "use", "van", "vat", "vet", "via", "vie",
		"wag", "war", "was", "wax", "way", "web", "wed", "wet", "who", "why",
		"wig", "win", "wit", "woe", "wok", "won", "woo", "wow", "yak", "yam",
		"yap", "yaw", "yea", "yes", "yet", "yew", "you", "zap", "zen", "zip",
		// 4-letter words
		"able", "also", "area", "back", "ball", "bank", "base", "bear", "beat", "been",
		"bell", "best", "bird", "blue", "boat", "body", "book", "born", "both", "came",
		"card", "care", "case", "city", "come", "cost", "dark", "date", "days", "deal",
		"deep", "does", "done", "door", "down", "draw", "drop", "each", "east", "easy",
		"edge", "else", "even", "ever", "eyes", "face", "fact", "fall", "farm", "fast",
		"fear", "feel", "feet", "felt", "file", "fill", "film", "find", "fine", "fire",
		"fish", "five", "food", "foot", "form", "four", "free", "from", "full", "game",
		"gave", "girl", "give", "glad", "goes", "gold", "gone", "good", "grow", "hair",
		"half", "hall", "hand", "hard", "have", "head", "hear", "heat", "help", "here",
		"high", "hill", "hold", "home", "hope", "hour", "idea", "into", "just", "keep",
		"kept", "kind", "king", "knew", "know", "lack", "lady", "lake", "land", "last",
		"late", "lead", "left", "less", "life", "like", "line", "list", "live", "long",
		"look", "lord", "lose", "loss", "lost", "love", "made", "main", "make", "many",
		"mark", "mass", "meet", "mind", "miss", "more", "most", "move", "much", "must",
		"name", "near", "need", "news", "next", "nice", "note", "once", "only", "open",
		"over", "page", "paid", "part", "pass", "past", "pick", "plan", "play", "plus",
		"pool", "poor", "post", "pull", "race", "rain", "rate", "read", "real", "rest",
		"rich", "ride", "rise", "road", "rock", "role", "room", "rule", "safe", "said",
		"sale", "same", "save", "says", "seat", "seem", "seen", "self", "sell", "send",
		"sent", "ship", "shop", "shot", "show", "shut", "side", "sign", "size", "slow",
		"snow", "sold", "some", "song", "soon", "sort", "spot", "star", "stay", "step",
		"stop", "such", "sure", "take", "talk", "tell", "term", "test", "than", "that",
		"them", "then", "they", "this", "thus", "time", "told", "took", "town", "tree",
		"trip", "true", "turn", "type", "unit", "upon", "used", "very", "view", "vote",
		"wait", "walk", "wall", "want", "ward", "warm", "ways", "week", "well", "went",
		"were", "west", "what", "when", "whom", "wide", "wife", "will", "wind", "wish",
		"with", "word", "work", "yard", "year", "your", "zero", "zone",
		// 5-letter words
		"about", "above", "added", "after", "again", "among", "began", "being", "black",
		"blood", "board", "bound", "bring", "build", "built", "carry", "cause", "child",
		"clear", "close", "comes", "could", "court", "cover", "death", "doing", "doubt",
		"early", "earth", "eight", "enemy", "enter", "equal", "every", "field", "fight",
		"final", "first", "floor", "force", "forms", "found", "front", "given", "going",
		"great", "green", "group", "hands", "happy", "heart", "heavy", "horse", "hotel",
		"hours", "house", "human", "ideas", "image", "known", "large", "later", "least",
		"leave", "level", "light", "lines", "lived", "local", "looks", "lower", "march",
		"means", "might", "money", "month", "moral", "moved", "music", "names", "never",
		"night", "north", "nothing", "offer", "often", "order", "other", "party", "peace",
		"place", "plain", "plant", "point", "power", "press", "price", "range", "reach",
		"ready", "right", "river", "round", "rules", "seems", "sense", "serve", "shall",
		"share", "short", "shown", "since", "small", "sound", "south", "space", "speak",
		"spend", "stage", "stand", "start", "state", "still", "stock", "stone", "stood",
		"story", "study", "style", "sweet", "table", "taken", "terms", "their", "there",
		"these", "thing", "think", "third", "those", "three", "times", "today", "total",
		"trade", "train", "trees", "trial", "tried", "truth", "under", "union", "until",
		"value", "voice", "water", "weeks", "where", "which", "while", "white", "whole",
		"whose", "woman", "words", "world", "would", "write", "wrong", "years", "young",
	}
	return t.DictionaryService.LoadWords(words)
}
