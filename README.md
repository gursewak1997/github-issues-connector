# GitHub Issues Connector

A **Redpanda Connect-style streaming pipeline** that automatically fetches GitHub issues and transforms them with AI-powered summarization.

> **What it does**: Fetches issues from GitHub → Streams to Redpanda → AI summarizes → Streams summaries back to Redpanda

## Architecture

This project follows a **Redpanda Connect-style architecture** with separate concerns for scalability and maintainability:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   GitHub API    │───▶│    Producer     │───▶│   Redpanda      │
│                 │    │                 │    │  github-issues  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                         │
                                                         ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Redpanda      │◀───│   Consumer +    │◀───│   Redpanda      │
│github-issues-   │    │   AI Transformer│    │  github-issues  │
│summarized       │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Producer (`producer/`)
- **Purpose**: Data ingestion from GitHub
- **Behavior**: Fetches issues from `bootc-dev/bootc` (past 24 hours)
- **Output**: Raw issues to Redpanda topic `github-issues`
- **Usage**: Run as cron job or standalone program

### Consumer + Transformer (`consumer/`)
- **Purpose**: AI-powered data transformation
- **Behavior**: Continuously processes raw issues with OpenAI
- **Input**: Consumes from `github-issues` topic
- **Output**: AI summaries to `github-issues-summarized` topic
- **Usage**: Long-running streaming service

## Quick Start

### Producer (Data Ingestion)
```bash
cd producer
go mod download
export GITHUB_TOKEN=your_token_here  # Optional, for higher rate limits
go run main.go
```

**What happens**: Fetches issues from GitHub → Produces to Redpanda topic `github-issues`

### Consumer + Transformer (AI Processing)
```bash
cd consumer
go mod download
export OPENAI_API_KEY=your_openai_key_here
go run main.go
```

**What happens**: Consumes from `github-issues` → AI summarizes → Produces to `github-issues-summarized`

---

**Pro Tip**: Run the producer first to populate data, then start the consumer to process it!

## Requirements

| Component | Requirement | Notes |
|-----------|-------------|-------|
| **Go** | 1.21+ | Both producer and consumer |
| **Redpanda** | Running on `localhost:9092` | Streaming platform |
| **OpenAI API** | Valid API key | Required for consumer (AI summarization) |
| **GitHub Token** | Valid token | Optional for producer (higher rate limits) |

## Configuration

### Producer Configuration
Edit constants in `producer/main.go`:

| Setting | Current Value | Description |
|---------|---------------|-------------|
| `repoOwner` | `bootc-dev` | GitHub repository owner |
| `repoName` | `bootc` | GitHub repository name |
| `brokerAddress` | `localhost:9092` | Redpanda broker address |
| `topic` | `github-issues` | Output topic name |

### Consumer Configuration
Edit constants in `consumer/main.go`:

| Setting | Current Value | Description |
|---------|---------------|-------------|
| `sourceTopic` | `github-issues` | Input topic to consume from |
| `summarizedTopic` | `github-issues-summarized` | Output topic for summaries |
| `brokerAddress` | `localhost:9092` | Redpanda broker address |
| `consumerGroupID` | `github-issues-summarizer` | Consumer group identifier |

## Output Topics

The pipeline produces data to two Redpanda topics:

### `github-issues` - Raw Issues (Producer Output)
Contains the complete GitHub issue data as fetched from the API:

```json
{
  "id": 12345,
  "number": 42,
  "title": "Issue Title",
  "body": "Full issue description...",
  "html_url": "https://github.com/bootc-dev/bootc/issues/42"
}
```

### `github-issues-summarized` - AI Summarized Issues (Consumer Output)
Contains AI-generated summaries with enhanced metadata:

```json
{
  "id": 12345,
  "number": 42,
  "title": "Issue Title",
  "summary": "AI-generated 2-3 sentence summary",
  "html_url": "https://github.com/bootc-dev/bootc/issues/42",
  "original_body": "Full issue description...",
  "processed_at": "2024-01-15T10:30:00Z"
}
```

## Deployment

### Producer (Batch Job)
- **Manual execution**: `go run producer/main.go`
- **Cron job**: `0 */6 * * * cd /path/to/producer && go run main.go` (every 6 hours)
- **Docker**: Containerized for scheduled execution
- **Use case**: Data ingestion at regular intervals

### Consumer (Streaming Service)
- **Long-running service**: `go run consumer/main.go`
- **Systemd**: Production service deployment
- **Docker**: Container with health checks
- **Kubernetes**: Scalable deployment
- **Use case**: Continuous data processing

---

**Typical Workflow**:
1. **Producer runs** every 6 hours → Fetches new issues → Produces to `github-issues`
2. **Consumer runs continuously** → Processes new issues → Produces summaries to `github-issues-summarized`
3. **Real-time insights** → Get AI summaries as soon as new issues arrive!

## Use Cases

- **Issue Monitoring**: Track new GitHub issues in real-time
- **AI Insights**: Get intelligent summaries for quick issue assessment
- **Data Pipeline**: Feed summarized data to dashboards, analytics, or other systems
- **Team Productivity**: Quickly understand issue context without reading full descriptions
- **Streaming Analytics**: Real-time processing of GitHub activity

## Troubleshooting

- **Producer fails**: Check GitHub API rate limits and token validity
- **Consumer fails**: Verify OpenAI API key and Redpanda connectivity
- **No data flow**: Ensure Redpanda is running on `localhost:9092`
- **AI errors**: Check OpenAI API quota and response format
