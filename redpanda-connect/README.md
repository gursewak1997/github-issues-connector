# Redpanda Connect Implementation

This directory contains the Redpanda Connect configuration files for implementing the GitHub Issues pipeline.

## Overview

Instead of custom Go code, we use Redpanda Connect's built-in connectors:
- **GitHub Source Connector**: Fetches issues from GitHub
- **AI Transformer**: Uses OpenAI to summarize issues

## Files

- `github-source-connector.json` - Connector to fetch GitHub issues
- `github-issues-ai-transformer.json` - Connector to transform issues with AI

## Setup

### 1. Install Redpanda Connect
```bash
# Download Redpanda Connect
wget https://packages.confluent.io/maven/io/confluent/kafka-connect-github/latest/kafka-connect-github-latest.jar

# Place in Redpanda Connect plugins directory
cp kafka-connect-github-latest.jar /path/to/redpanda-connect/plugins/
```

### 2. Configure Environment Variables
```bash
export OPENAI_API_KEY=your_openai_api_key_here
export GITHUB_TOKEN=your_github_token_here  # Optional, for higher rate limits
```

### 3. Deploy Connectors

#### Deploy GitHub Source
```bash
curl -X POST -H "Content-Type: application/json" \
  --data @github-source-connector.json \
  http://localhost:8083/connectors
```

#### Deploy AI Transformer
```bash
curl -X POST -H "Content-Type: application/json" \
  --data @github-issues-ai-transformer.json \
  http://localhost:8083/connectors
```

## Monitor Connectors

### Check Status
```bash
# List all connectors
curl http://localhost:8083/connectors

# Check specific connector status
curl http://localhost:8083/connectors/github-issues-source/status
curl http://localhost:8083/connectors/github-issues-ai-transformer/status
```

### View Logs
```bash
# Check connector logs
curl http://localhost:8083/connectors/github-issues-source/logs
```

## Benefits of This Approach

✅ **No custom code** - Pure configuration
✅ **Built-in monitoring** - Connector health and metrics
✅ **Automatic scaling** - Handles backpressure
✅ **Production ready** - Enterprise-grade reliability
✅ **Easy maintenance** - Update config, restart connector
✅ **Built-in error handling** - Dead letter queues, retries

## Data Flow

```
GitHub API → GitHub Source Connector → github-issues topic → AI Transformer → github-issues-summarized topic
```

## Troubleshooting

- **Connector fails**: Check logs with `/status` endpoint
- **No data**: Verify GitHub token and repository access
- **AI errors**: Check OpenAI API key and quota
- **Connection issues**: Ensure Redpanda is running on port 9092 