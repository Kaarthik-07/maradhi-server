package handler

// ai.go — Claude (Anthropic) powered features for Maradhi.
//
// These endpoints call the Anthropic API directly using net/http — no SDK.
// We keep it dependency-free: one file, one function per feature.
//
// Routes (only registered when ANTHROPIC_API_KEY is set):
//   POST /api/v1/ai/suggest    → suggests tasks based on user context
//   POST /api/v1/ai/summarise  → daily summary of tasks + mood + habits
//
// The API key lives in config (loaded from ANTHROPIC_API_KEY env var).
// Handlers in this file never touch the DB directly — they read context
// from the request body and call Claude.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const anthropicURL = "https://api.anthropic.com/v1/messages"
const claudeModel  = "claude-sonnet-4-5"

// ─── Request / Response types ─────────────────────────────────────────────────

type AISuggestRequest struct {
	// Context the mobile app sends — what the user is working on today
	CurrentTasks []string `json:"current_tasks"` // titles of today's tasks
	RecentMoods  []string `json:"recent_moods"`  // last 3 mood entries
	Goal         string   `json:"goal"`          // optional: "I want to be more productive"
}

type AISuggestResponse struct {
	Suggestions []TaskSuggestion `json:"suggestions"`
}

type TaskSuggestion struct {
	Title    string `json:"title"`
	Priority string `json:"priority"` // high | medium | low
	Reason   string `json:"reason"`
}

type AISummariseRequest struct {
	CompletedTasks  []string `json:"completed_tasks"`
	PendingTasks    []string `json:"pending_tasks"`
	HabitsCompleted []string `json:"habits_completed"`
	TodayMood       string   `json:"today_mood"`
}

type AISummariseResponse struct {
	Summary     string   `json:"summary"`
	WinOfTheDay string   `json:"win_of_the_day"`
	Suggestions []string `json:"suggestions"`
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// AISuggestTasks asks Claude to suggest new tasks based on the user's context.
//
// POST /api/v1/ai/suggest
// Body: AISuggestRequest
func (h *Handler) AISuggestTasks(w http.ResponseWriter, r *http.Request) {
	var req AISuggestRequest
	if !decode(w, r, &req) {
		return
	}

	prompt := buildSuggestPrompt(req)

	raw, err := callClaude(h.cfg.AnthropicAPIKey, prompt, 600)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI service unavailable: "+err.Error())
		return
	}

	// Claude is instructed to respond in JSON — parse it directly
	var result AISuggestResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		// Fallback: return raw text if JSON parse fails
		writeJSON(w, http.StatusOK, map[string]string{"raw": raw})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// AISummariseDay asks Claude to summarise the user's day.
//
// POST /api/v1/ai/summarise
// Body: AISummariseRequest
func (h *Handler) AISummariseDay(w http.ResponseWriter, r *http.Request) {
	var req AISummariseRequest
	if !decode(w, r, &req) {
		return
	}

	prompt := buildSummarisePrompt(req)

	raw, err := callClaude(h.cfg.AnthropicAPIKey, prompt, 500)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI service unavailable: "+err.Error())
		return
	}

	var result AISummariseResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"raw": raw})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ─── Prompt builders ──────────────────────────────────────────────────────────

func buildSuggestPrompt(req AISuggestRequest) string {
	var b strings.Builder
	b.WriteString("You are a productivity assistant inside a personal task manager app called Maradhi.\n\n")
	b.WriteString("The user's current tasks today:\n")
	for _, t := range req.CurrentTasks {
		b.WriteString("- " + t + "\n")
	}
	if len(req.RecentMoods) > 0 {
		b.WriteString("\nRecent mood: " + strings.Join(req.RecentMoods, ", ") + "\n")
	}
	if req.Goal != "" {
		b.WriteString("\nUser's goal: " + req.Goal + "\n")
	}
	b.WriteString(`
Suggest 3 additional tasks that would complement their current work and help them have a productive day.
Be specific and actionable. Consider their mood when setting priority.

Respond ONLY with valid JSON in this exact shape, no explanation, no markdown:
{
  "suggestions": [
    { "title": "...", "priority": "high|medium|low", "reason": "one sentence why" },
    { "title": "...", "priority": "high|medium|low", "reason": "one sentence why" },
    { "title": "...", "priority": "high|medium|low", "reason": "one sentence why" }
  ]
}`)
	return b.String()
}

func buildSummarisePrompt(req AISummariseRequest) string {
	var b strings.Builder
	b.WriteString("You are a supportive productivity coach inside the Maradhi app.\n\n")
	fmt.Fprintf(&b, "Today's mood: %s\n", req.TodayMood)
	if len(req.CompletedTasks) > 0 {
		b.WriteString("Completed tasks:\n")
		for _, t := range req.CompletedTasks {
			b.WriteString("- " + t + "\n")
		}
	}
	if len(req.PendingTasks) > 0 {
		b.WriteString("Still pending:\n")
		for _, t := range req.PendingTasks {
			b.WriteString("- " + t + "\n")
		}
	}
	if len(req.HabitsCompleted) > 0 {
		b.WriteString("Habits completed: " + strings.Join(req.HabitsCompleted, ", ") + "\n")
	}
	b.WriteString(`
Write a brief, warm end-of-day summary. Be encouraging but honest.

Respond ONLY with valid JSON, no markdown:
{
  "summary": "2-3 sentence summary of their day",
  "win_of_the_day": "the single best thing they did today",
  "suggestions": ["one thing for tomorrow", "one habit suggestion"]
}`)
	return b.String()
}

// ─── Anthropic API client ─────────────────────────────────────────────────────

// callClaude sends a single user message to Claude and returns the text response.
// We use net/http directly — no Anthropic SDK — to stay dependency-free.
func callClaude(apiKey, userMessage string, maxTokens int) (string, error) {
	// Anthropic messages API payload
	body, err := json.Marshal(map[string]any{
		"model":      claudeModel,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": userMessage},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, anthropicURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic API %d: %s", resp.StatusCode, string(b))
	}

	// Parse the Anthropic response envelope
	var envelope struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	for _, block := range envelope.Content {
		if block.Type == "text" {
			return strings.TrimSpace(block.Text), nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}
