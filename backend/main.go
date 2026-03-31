package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"health-assistant/backend/agent"
	"health-assistant/backend/handlers"
	"health-assistant/backend/jobs"
	"health-assistant/backend/llm"
	"health-assistant/backend/memory"
	"health-assistant/backend/tools"
)

func main() {
	loadEnvFile(".env")
	loadEnvFile(filepath.Join("..", ".env"))

	apiKey := os.Getenv("GEMINI_API_KEY")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := tools.LoadUserProfile("data/user_profile.json"); err != nil {
		log.Printf("warning: user profile preload failed: %v", err)
	}

	store := memory.NewStore()
	memoryTool := tools.NewMemoryTool(store)
	embedder := memory.NewEmbeddingClient(apiKey)
	geminiClient := llm.NewGeminiClient(apiKey, os.Getenv("GEMINI_MODEL"))

	workerPool := jobs.NewWorkerPool(4)
	workerPool.Start()

	a := agent.NewAgent(geminiClient, embedder, memoryTool, workerPool)
	chatHandler := handlers.NewChatHandler(a, workerPool)

	http.Handle("/api/chat", chatHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := fmt.Sprintf(":%s", port)
	log.Printf("backend listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if k == "" {
			continue
		}

		if _, exists := os.LookupEnv(k); !exists {
			_ = os.Setenv(k, v)
		}
	}
}
