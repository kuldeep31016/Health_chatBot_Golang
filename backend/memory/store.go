package memory

import (
	"math"
	"sort"
	"sync"
)

type MemoryEntry struct {
	Text   string
	Vector []float64
}

type Store struct {
	mu      sync.RWMutex
	entries []MemoryEntry
}

func NewStore() *Store {
	return &Store{entries: make([]MemoryEntry, 0)}
}

func (s *Store) Add(entry MemoryEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
}

func (s *Store) TopK(query []float64, topK int) []MemoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if topK <= 0 || len(s.entries) == 0 {
		return nil
	}

	type scored struct {
		entry MemoryEntry
		score float64
	}

	scoredEntries := make([]scored, 0, len(s.entries))
	for _, e := range s.entries {
		scoredEntries = append(scoredEntries, scored{
			entry: e,
			score: CosineSimilarity(query, e.Vector),
		})
	}

	sort.Slice(scoredEntries, func(i, j int) bool {
		return scoredEntries[i].score > scoredEntries[j].score
	})

	if topK > len(scoredEntries) {
		topK = len(scoredEntries)
	}

	out := make([]MemoryEntry, 0, topK)
	for i := 0; i < topK; i++ {
		out = append(out, scoredEntries[i].entry)
	}

	return out
}

func CosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	var dot float64
	var magA float64
	var magB float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}

	if magA == 0 || magB == 0 {
		return 0
	}

	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}
