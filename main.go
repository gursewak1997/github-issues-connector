package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	topic         = "github-issues"
	brokerAddress = "localhost:9092"
	// You can change these to the repository you want to monitor
	repoOwner = "bootc-dev"
	repoName  = "bootc"
)

type Issue struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	URL    string `json:"html_url"`
}

func fetchIssues(since time.Time) ([]Issue, error) {
	client := &http.Client{}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?since=%s&state=open", repoOwner, repoName, since.Format(time.RFC3339))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Optional: Use GITHUB_TOKEN env var if set for authentication (to increase rate limits)
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

func main() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Track latest time so we only fetch new issues each time
	lastFetchTime := time.Now().Add(-24 * time.Hour)

	for range ticker.C {
		log.Println("Fetching issues from GitHub...")
		issues, err := fetchIssues(lastFetchTime)
		if err != nil {
			log.Println("Error fetching issues:", err)
			continue
		}
		if len(issues) > 0 {
			err = produceIssuesToKafka(issues)
			if err != nil {
				log.Println("Error producing to Kafka:", err)
				continue
			}
			// Update lastFetchTime to the latest issue updated_at
			lastFetchTime = time.Now()
		} else {
			log.Println("No new issues found.")
		}
	}
}
