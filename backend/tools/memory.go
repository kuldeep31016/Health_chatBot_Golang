package tools

import "health-assistant/backend/memory"

type MemoryEntry = memory.MemoryEntry

type MemoryTool struct {
	Store *memory.Store
}

func NewMemoryTool(store *memory.Store) *MemoryTool {
	return &MemoryTool{Store: store}
}

func (m *MemoryTool) StoreMemory(text string, vector []float64) {
	if m == nil || m.Store == nil || len(vector) == 0 || text == "" {
		return
	}
	m.Store.Add(memory.MemoryEntry{Text: text, Vector: vector})
}

func (m *MemoryTool) RetrieveRelevantMemory(queryVector []float64, topK int) []MemoryEntry {
	if m == nil || m.Store == nil || len(queryVector) == 0 {
		return nil
	}
	return m.Store.TopK(queryVector, topK)
}
