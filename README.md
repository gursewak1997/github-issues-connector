# GitHub Issues Connector

A streaming data pipeline that fetches GitHub issues and transforms them with AI-powered summarization using Redpanda Connect.

**Data Flow**: GitHub → Redpanda → AI Summarization → Redpanda

## Architecture

```
GitHub API → GitHub Source Connector → github-issues topic → AI Transformer → github-issues-summarized topic
```

- **GitHub Source**: Fetches issues from `bootc-dev/bootc` (past 24h)
- **AI Transformer**: Uses OpenAI to generate summaries
- **Topics**: `github-issues` (raw), `github-issues-summarized` (AI processed)

## Quick Start

```bash
# Deploy connectors
curl -X POST -H "Content-Type: application/json" \
  --data @redpanda-connect/github-source-connector.json \
  http://localhost:8083/connectors

curl -X POST -H "Content-Type: application/json" \
  --data @redpanda-connect/github-issues-ai-transformer.json \
  http://localhost:8083/connectors
```

**Setup**: See `redpanda-connect/README.md` for full instructions.

## Implementation Comparison

| Aspect | Redpanda Connect | Custom Go |
|--------|------------------|-----------|
| **Setup** | Configuration files | Code compilation |
| **Maintenance** | Update config, restart | Modify code, rebuild |
| **Monitoring** | Built-in metrics | Custom logging |
| **Reliability** | Production-tested | Depends on code quality |
| **Production Ready** | Yes | Requires testing |
| **Deployment** | REST API calls | Binary deployment |

## Requirements

- **Redpanda**: Running on `localhost:9092`
- **OpenAI API**: Valid API key for AI summarization
- **GitHub Token**: Optional, for higher rate limits

## Configuration

Edit JSON files in `redpanda-connect/` to change:
- Repository owner/name
- Redpanda broker address
- Topic names
- Polling intervals

## Output Topics

- **`github-issues`**: Raw GitHub issue data
- **`github-issues-summarized`**: AI-generated summaries with metadata

## Deployment

- **GitHub Source**: Polls GitHub API every hour
- **AI Transformer**: Processes issues continuously
- **REST API**: Deploy via HTTP endpoints
- **Production**: Docker, Kubernetes, monitoring included

## Use Cases

- **Issue Monitoring**: Real-time GitHub issue tracking
- **AI Insights**: Quick issue assessment with summaries
- **Data Pipeline**: Feed to dashboards and analytics
- **Team Productivity**: Understand context quickly
