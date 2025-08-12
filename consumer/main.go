package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	sourceTopic     = "github-issues"
	summarizedTopic = "github-issues-summarized"
	brokerAddress   = "localhost:9092"
	consumerGroupID = "github-issues-summarizer"
)

type Issue struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	URL    string `json:"html_url"`
}

type SummarizedIssue struct {
	ID           int    `json:"id"`
	Number       int    `json:"number"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	URL          string `json:"html_url"`
	OriginalBody string `json:"original_body"`
	ProcessedAt  string `json:"processed_at"`
}

func summarizeWithAI(issue Issue) (SummarizedIssue, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return SummarizedIssue{}, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Prepare the prompt for summarization
	prompt := fmt.Sprintf(`Summarize this GitHub issue in 2-3 sentences:

Title: %s
Body: %s

Summary:`, issue.Title, issue.Body)

	// Create OpenAI API request
	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  150,
		"temperature": 0.3,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return SummarizedIssue{}, err
	}

	// Make API call to OpenAI
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return SummarizedIssue{}, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return SummarizedIssue{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SummarizedIssue{}, err
	}

	if resp.StatusCode != 200 {
		return SummarizedIssue{}, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	// Parse OpenAI response
	var openAIResp map[string]interface{}
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return SummarizedIssue{}, err
	}

	choices, ok := openAIResp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return SummarizedIssue{}, fmt.Errorf("invalid OpenAI response format")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return SummarizedIssue{}, fmt.Errorf("invalid choice format")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return SummarizedIssue{}, fmt.Errorf("invalid message format")
	}

	summary, ok := message["content"].(string)
	if !ok {
		return SummarizedIssue{}, fmt.Errorf("invalid content format")
	}

	return SummarizedIssue{
		ID:           issue.ID,
		Number:       issue.Number,
		Title:        issue.Title,
		Summary:      summary,
		URL:          issue.URL,
		OriginalBody: issue.Body,
		ProcessedAt:  time.Now().Format(time.RFC3339),
	}, nil
}

func produceSummarizedIssueToRedpanda(summarizedIssue SummarizedIssue) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{brokerAddress},
		Topic:    summarizedTopic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	data, err := json.Marshal(summarizedIssue)
	if err != nil {
		return fmt.Errorf("failed to marshal summarized issue %d: %v", summarizedIssue.Number, err)
	}

	err = writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(fmt.Sprintf("%d", summarizedIssue.ID)),
			Value: data,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write summarized issue %d to Redpanda: %v", summarizedIssue.Number, err)
	}

	log.Printf("Produced summarized issue #%d to Redpanda topic: %s", summarizedIssue.Number, summarizedTopic)
	return nil
}

func processMessage(ctx context.Context, msg kafka.Message) error {
	var issue Issue
	if err := json.Unmarshal(msg.Value, &issue); err != nil {
		return fmt.Errorf("failed to unmarshal issue: %v", err)
	}

	log.Printf("Processing issue #%d: %s", issue.Number, issue.Title)

	// Summarize with AI
	summarizedIssue, err := summarizeWithAI(issue)
	if err != nil {
		return fmt.Errorf("failed to summarize issue #%d: %v", issue.Number, err)
	}

	// Produce to summarized topic
	if err := produceSummarizedIssueToRedpanda(summarizedIssue); err != nil {
		return fmt.Errorf("failed to produce summarized issue #%d: %v", issue.Number, err)
	}

	return nil
}

func main() {
	log.Println("Starting GitHub Issues Consumer + AI Transformer...")
	log.Printf("Consuming from topic: %s", sourceTopic)
	log.Printf("Producing to topic: %s", summarizedTopic)

	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    sourceTopic,
		GroupID:  consumerGroupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer reader.Close()

	ctx := context.Background()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			continue
		}

		if err := processMessage(ctx, msg); err != nil {
			log.Printf("Error processing message: %v", err)
			// Continue processing other messages
			continue
		}

		// Commit the message
		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Error committing message: %v", err)
		}
	}
}
