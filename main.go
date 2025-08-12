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
	topic           = "github-issues"
	summarizedTopic = "github-issues-summarized"
	brokerAddress   = "localhost:9092"
	repoOwner       = "bootc-dev"
	repoName        = "bootc"
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
}

func fetchIssues(since time.Time) ([]Issue, error) {
	client := &http.Client{}
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/issues?since=%s&state=open",
		repoOwner, repoName, since.Format(time.RFC3339),
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, err
	}
	return issues, nil
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
	}, nil
}

func produceIssuesToKafka(issues []Issue) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{brokerAddress},
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	for _, issue := range issues {
		data, err := json.Marshal(issue)
		if err != nil {
			log.Printf("Failed to marshal issue %d: %v", issue.Number, err)
			continue
		}

		err = writer.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(fmt.Sprintf("%d", issue.ID)),
				Value: data,
			},
		)
		if err != nil {
			log.Printf("Failed to write issue %d to Kafka: %v", issue.Number, err)
		} else {
			log.Printf("Produced issue #%d to Kafka", issue.Number)
		}
	}
	return nil
}

func produceSummarizedIssuesToKafka(summarizedIssues []SummarizedIssue) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{brokerAddress},
		Topic:    summarizedTopic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	for _, issue := range summarizedIssues {
		data, err := json.Marshal(issue)
		if err != nil {
			log.Printf("Failed to marshal summarized issue %d: %v", issue.Number, err)
			continue
		}

		err = writer.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(fmt.Sprintf("%d", issue.ID)),
				Value: data,
			},
		)
		if err != nil {
			log.Printf("Failed to write summarized issue %d to Kafka: %v", issue.Number, err)
		} else {
			log.Printf("Produced summarized issue #%d to Kafka", issue.Number)
		}
	}
	return nil
}

func main() {
	// Fetch issues from the past 24h
	lastFetchTime := time.Now().Add(-24 * time.Hour)

	log.Println("Fetching issues from GitHub...")
	issues, err := fetchIssues(lastFetchTime)
	if err != nil {
		log.Fatal("Error fetching issues:", err)
	}

	if len(issues) > 0 {
		// Produce original issues to Kafka
		if err := produceIssuesToKafka(issues); err != nil {
			log.Fatal("Error producing original issues to Kafka:", err)
		}
		log.Println("Done producing original issues.")

		// Summarize issues with AI
		log.Println("Summarizing issues with AI...")
		var summarizedIssues []SummarizedIssue
		for _, issue := range issues {
			summarized, err := summarizeWithAI(issue)
			if err != nil {
				log.Printf("Failed to summarize issue #%d: %v", issue.Number, err)
				continue
			}
			summarizedIssues = append(summarizedIssues, summarized)
			log.Printf("Summarized issue #%d", issue.Number)
		}

		// Produce summarized issues to Kafka
		if len(summarizedIssues) > 0 {
			if err := produceSummarizedIssuesToKafka(summarizedIssues); err != nil {
				log.Fatal("Error producing summarized issues to Kafka:", err)
			}
			log.Printf("Done producing %d summarized issues.", len(summarizedIssues))
		}
	} else {
		log.Println("No new issues found.")
	}
}
