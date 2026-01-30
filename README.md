# bedrock-tools

Go examples for working with AWS Bedrock, demonstrating text embeddings and conversational AI with tool use.

## What's Included

This repository contains two example applications:

1. **Text Embedding** - Convert text to vector embeddings using Amazon Titan
2. **Tool Use** - Multi-turn conversations with AI models that can call functions

## Prerequisites

- Go 1.25 or later
- AWS account with Bedrock access enabled
- AWS credentials configured locally (`~/.aws/credentials` or environment variables)

## Installation

```bash
go mod download
```

## Usage

### Text Embeddings

Generate vector embeddings for text:

```bash
go run cmd/embed/main.go
```

This converts "hello world" into a 1024-dimensional vector using Amazon Titan Embed Text v2. The output includes the embedding array and input token count.

### Conversational AI with Tools

Run a multi-turn conversation where the AI can invoke tools:

```bash
go run cmd/tool-use/main.go
```

This demonstrates an agentic pattern where the AI answers questions by calling available functions:
- `get_user` - Retrieve user information by ID
- `get_user_orders` - Get order history for a user
- `eval_js` - Execute JavaScript code in a sandboxed environment

The conversation runs for up to 10 turns, with the AI deciding which tools to call based on the user's query.

## Configuration

### AWS Region

Both examples default to `us-west-2`. To use a different region, modify the `cfg` initialization:

```go
cfg, err := config.LoadDefaultConfig(context.Background(),
    config.WithRegion("your-region"))
```

### Model Selection

The tool-use example supports multiple models including GPT-OSS and Qwen variants. Change the `modelID` variable in `cmd/tool-use/main.go`:

```go
modelID := "your-model-id-here"
```

### Embedding Dimensions

Adjust output dimensions in `cmd/embed/main.go`:

```go
embeddingConfig := map[string]any{
    "outputEmbeddingLength": 1024,  // 256, 512, or 1024
    // ...
}
```

## How It Works

### Embeddings

The embedding example sends text to Bedrock's `InvokeModel` API with the Titan Embed model. The response contains a vector representation useful for semantic search, clustering, or similarity comparisons.

### Tool Use

The tool-use example implements an agentic loop:

1. Send a message to the AI model with available tool definitions
2. The model responds with either text or tool invocation requests
3. Execute requested tools locally and return results
4. Continue conversation until the model provides a final answer

Tools are defined with JSON Schema, allowing the model to understand their parameters and when to use them.

## Project Structure

```
bedrock-tools/
├── cmd/
│   ├── embed/main.go       # Embedding example
│   └── tool-use/main.go    # Conversational AI with tools
├── go.mod                  # Dependencies
├── go.sum                  # Dependency checksums
└── devbox.json            # Development environment
```

## Dependencies

- AWS SDK for Go v2 - Bedrock API integration
- goja - JavaScript runtime for sandboxed code execution
- lo - Utility functions

## License

See LICENSE file for details.

## Additional Resources

- [AWS Bedrock Documentation](https://docs.aws.amazon.com/bedrock/)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
