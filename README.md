# GitHub Issues Connector

A Go application that monitors a GitHub repository for new issues and streams them to Apache Kafka.

## What it does

- Fetches open issues from `bootc-dev/bootc` every minute
- Publishes issues to Kafka topic `github-issues`
- Runs continuously with incremental fetching

## Quick start

1. Install dependencies: `go mod download`
2. (Optional) Set `GITHUB_TOKEN` for higher rate limits
3. Run: `go run main.go`

## Requirements

- Go 1.16+
- Kafka running on `localhost:9092`

## Configuration

Edit constants in `main.go` to change:
- Repository owner/name
- Kafka broker address
- Fetch interval (currently 24 hours)
- Topic name
