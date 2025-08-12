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
	repoOwner     = "bootc-dev"
	repoName      = "bootc"
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

func produceIssuesToRedpanda(issues []Issue) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{brokerAddress},
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	defer func() {
		if err := writer.Close(); err != nil {
			log.Printf("Failed to close Kafka writer: %v", err)
		}
	}()

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
			log.Printf("Failed to write issue %d to Redpanda: %v", issue.Number, err)
		} else {
			log.Printf("Produced issue #%d to Redpanda", issue.Number)
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
		if err := produceIssuesToRedpanda(issues); err != nil {
			log.Fatal("Error producing to Redpanda:", err)
		}
		log.Printf("Successfully produced %d issues to Redpanda topic: %s", len(issues), topic)
	} else {
		log.Println("No new issues found.")
	}
}
