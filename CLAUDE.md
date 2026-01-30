# CLAUDE.md - AI Assistant Guide

This document provides context for AI assistants working on this codebase.

## Project Overview

This is a Go-based toolkit for interacting with AWS Bedrock, Amazon's generative AI platform. The project demonstrates two core capabilities:

1. **Text Embeddings** - Converting text to vector representations using Amazon Titan
2. **Tool Use** - Implementing agentic patterns with Claude models that can invoke functions

## Codebase Structure

```
bedrock-tools/
├── cmd/
│   ├── embed/          # Text embedding example
│   └── tool-use/       # Multi-turn conversation with tools
├── go.mod              # Go 1.25.5, AWS SDK v2, goja
└── devbox.json         # Development environment config
```

## Key Components

### cmd/embed/main.go
- Uses Amazon Titan Embed Text v2 model (`amazon.titan-embed-text-v2:0`)
- Converts text to 1024-dimensional embeddings
- Configuration: normalized output, TEXT embedding type
- Returns embeddings array and input token count

### cmd/tool-use/main.go
- Implements agentic loop (max 10 iterations)
- Supports multiple Claude models (GPT-OSS variants, Qwen)
- Three available tools:
  - `get_user(user_id)` - Mock user data retrieval
  - `get_user_orders(user_id)` - Mock order history
  - `eval_js(code)` - JavaScript execution in goja sandbox (1s timeout)
- Handles multi-turn conversations with tool invocations
- Tracks token usage across turns

## Development Patterns

### AWS Bedrock Integration
- Uses AWS SDK v2 for Go
- Requires AWS credentials configured in environment
- Direct API calls to `bedrockruntime` service
- JSON marshaling for request/response payloads

### Tool Schema Format
Tools are defined with:
- `toolSpec.name` - Function identifier
- `toolSpec.description` - Natural language description for the model
- `toolSpec.inputSchema` - JSON Schema for parameters

### Error Handling
- Tool execution failures return error messages to the model
- JavaScript execution has timeout protection (1s)
- Failed tool calls don't break the conversation loop

## Working with This Codebase

### Making Changes
- Both cmd packages are standalone executables
- Tool definitions are in-line in `tool-use/main.go`
- Mock data is hardcoded (see user/order data structures)
- Model IDs are configurable via variables

### Testing Locally
- Requires AWS credentials with Bedrock access
- Uses us-west-2 region by default
- Run with `go run cmd/embed/main.go` or `go run cmd/tool-use/main.go`

### Common Modifications
- **Add new tools**: Extend the `tools` array with new ToolConfiguration entries
- **Change models**: Modify `modelID` variable in main.go
- **Adjust parameters**: Update InferenceConfiguration fields (maxTokens, temperature, etc.)
- **Extend mock data**: Add more user/order entries to the maps

## Dependencies to Know

- `github.com/aws/aws-sdk-go-v2` - AWS service clients
- `github.com/dop251/goja` - JavaScript VM for sandboxed execution
- `github.com/samber/lo` - Functional utilities (Map, etc.)

## Security Notes

- JavaScript sandbox prevents file I/O and network access
- 1-second timeout prevents infinite loops
- No validation on user_id inputs (mock data only)
- AWS credentials inherited from environment
