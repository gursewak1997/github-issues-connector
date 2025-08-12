# GitHub Issues Connector

A Go application that monitors a GitHub repository for new issues and streams them to Apache Kafka.

## What it does

- Fetches open issues from `bootc-dev/bootc` from the past 24 hours
- Publishes original issues to Redpanda topic `github-issues`
- Uses AI (OpenAI) to generate concise summaries of each issue
- Publishes summarized issues to Redpanda topic `github-issues-summarized`
- Runs once and exits (not continuous)

## Quick start

1. Install dependencies: `go mod download`
2. Set `OPENAI_API_KEY` for AI summarization
3. (Optional) Set `GITHUB_TOKEN` for higher rate limits
4. Run: `go run main.go`

## Requirements

- Go 1.16+
- Redpanda running on `localhost:9092`
- OpenAI API key for AI summarization

## Configuration

Edit constants in `main.go` to change:
- Repository owner/name (currently `bootc-dev/bootc`)
- Kafka broker address
- Time range for fetching issues (currently 24 hours)
- Topic names (original and summarized)

## Output Topics

The connector produces data to two Redpanda topics:

### `github-issues` - Original Issues
Contains the complete GitHub issue data as fetched from the API.

### `github-issues-summarized` - AI Summarized Issues
Contains AI-generated summaries with the following structure:
- Issue metadata (ID, number, title, URL)
- AI-generated summary (2-3 sentences)
- Original body text for reference
