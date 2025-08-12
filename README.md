# GitHub Issues Connector

A Go application that monitors a GitHub repository for new issues and streams them to Apache Kafka.

## What it does

- Fetches open issues from `bootc-dev/bootc` from the past 24 hours
- Publishes issues to Kafka topic `github-issues`
- Runs once and exits (not continuous)

## Quick start

1. Install dependencies: `go mod download`
2. (Optional) Set `GITHUB_TOKEN` for higher rate limits
3. Run: `go run main.go`

## Requirements

- Go 1.16+
- Kafka running on `localhost:9092`

## Configuration

Edit constants in `main.go` to change:
- Repository owner/name (currently `bootc-dev/bootc`)
- Kafka broker address
- Time range for fetching issues (currently 24 hours)
- Topic name
