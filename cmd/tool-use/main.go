package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/dop251/goja"
	"github.com/samber/lo"
)

const (
	gptOss20B    = "openai.gpt-oss-20b-1:0"
	gptOss120B   = "openai.gpt-oss-120b-1:0"
	qwen3Next80B = "qwen.qwen3-next-80b-a3b"
)

func main() {
	ctx := context.Background()
	conf, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("failed to load aws config: %v", err)
		return
	}

	bd := bedrockruntime.NewFromConfig(conf)

	model := gptOss120B
	systemPrompt := "You are a service representative engaged in a polite conversation with a customer. " +
		"Anything you say to the user will be written to a simple chat interface, so respond with plain text. " +
		"Do not write any Markdown, code, or ASCII art. " +
		"Be as concise and straightforward as possible."
	systemPrompt = ""
	prompt := "How much do I usually spend on orders?"

	var inputTokens, outputTokens int
	defer func() {
		log.Printf("Usage: %d in %d out", inputTokens, outputTokens)
	}()

	toolConfig := &types.ToolConfiguration{
		Tools: []types.Tool{
			toolSpec(
				"get_user",
				"Get details of the user you are conversing with",
				toolDesc{},
			),
			toolSpec(
				"get_user_orders",
				"Get details of a user's orders",
				toolDesc{
					"user_id": requiredStringProp("ID of the user you are conversing with"),
				},
			),
			toolSpec(
				"eval_js",
				`Execute Javascript in a sandbox. `+
					`console.log() is not defined. Use log() or return a string to output text instead. `+
					`Use this tool to perform calculations. `+
					`Do not trust yourself to do arithmetic or accurately inspect strings.`,
				toolDesc{
					"script": requiredStringProp("Javascript to execute"),
				},
			),
		},
	}

	useTool := func(toolName string, params map[string]any) (toolResult map[string]any, toolErr error) {
		switch toolName {
		case "get_user":
			toolResult = map[string]any{
				"user_id": "ADFC15A01A81",
			}
		case "get_user_orders":
			if userID, ok := params["user_id"]; ok {
				if userID.(string) == "ADFC15A01A81" {
					toolResult = map[string]any{
						"orders": []map[string]any{
							{
								"order_id": "95B5342BA307",
								"total":    "$13.62",
							},
							{
								"order_id": "966937C77A5F",
								"total":    "$95.03",
							},
						},
					}
					return
				}
			}
			toolErr = fmt.Errorf("invalid user_id")
		case "eval_js":
			if _script, ok := params["script"]; ok {
				script := _script.(string)
				vm := goja.New()
				buf := new(strings.Builder)
				_ = vm.Set("log", func(call goja.FunctionCall) goja.Value {
					args := lo.Map(call.Arguments, func(arg goja.Value, _ int) string { return arg.ToString().String() })
					_, _ = fmt.Fprintf(buf, "%s\n", strings.Join(args, " "))
					return goja.Undefined()
				})
				time.AfterFunc(1*time.Second, func() {
					vm.Interrupt("deadline exceeded")
				})
				var scriptResult goja.Value
				scriptResult, toolErr = vm.RunString(script)
				if toolErr != nil {
					return
				}
				output := buf.String()
				if output == "" {
					output = scriptResult.String()
				}
				log.Printf("%q => %q", script, output)
				toolResult = map[string]any{
					"output": output,
				}
			}
		default:
			toolErr = fmt.Errorf("unknown tool %q", toolName)
		}
		return
	}

	messages := []types.Message{
		{
			Role: types.ConversationRoleUser,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{
					Value: prompt,
				},
			},
		},
	}

	for i := 0; i < 10; i++ {
		req := &bedrockruntime.ConverseInput{
			ModelId:    aws.String(model),
			Messages:   messages,
			ToolConfig: toolConfig,
		}
		if systemPrompt != "" {
			req.System = []types.SystemContentBlock{
				&types.SystemContentBlockMemberText{
					Value: systemPrompt,
				},
			}
		}
		var resp *bedrockruntime.ConverseOutput
		resp, err = bd.Converse(ctx, req)
		if err != nil {
			log.Printf("failed to call bedrock converse: %v", err)
			return
		}

		usage := lo.FromPtr(resp.Usage)
		inputTokens += int(lo.FromPtr(usage.InputTokens))
		outputTokens += int(lo.FromPtr(usage.OutputTokens))

		output := resp.Output.(*types.ConverseOutputMemberMessage)
		for _, block := range output.Value.Content {
			switch block.(type) {
			case *types.ContentBlockMemberText:
				text := block.(*types.ContentBlockMemberText)
				log.Println(text.Value)
				messages = append(messages, types.Message{
					Role: types.ConversationRoleAssistant,
					Content: []types.ContentBlock{
						&types.ContentBlockMemberText{
							Value: text.Value,
						},
					},
				})

			case *types.ContentBlockMemberReasoningContent:
				reasoning := block.(*types.ContentBlockMemberReasoningContent)
				if text, ok := reasoning.Value.(*types.ReasoningContentBlockMemberReasoningText); ok {
					log.Println("<think>")
					log.Println(lo.FromPtr(text.Value.Text))
					log.Println("</think>")
				}

			case *types.ContentBlockMemberToolUse:
				tool := block.(*types.ContentBlockMemberToolUse)
				toolUseID := lo.FromPtr(tool.Value.ToolUseId)
				toolName := lo.FromPtr(tool.Value.Name)
				var toolParams map[string]any
				_ = tool.Value.Input.UnmarshalSmithyDocument(&toolParams)
				log.Println(toolUseID, toolName, toolParams)

				messages = append(messages, types.Message{
					Role: types.ConversationRoleAssistant,
					Content: []types.ContentBlock{
						&types.ContentBlockMemberToolUse{
							Value: tool.Value,
						},
					},
				})

				toolResult, toolErr := useTool(toolName, toolParams)
				var toolResultBlock types.ToolResultBlock
				if toolErr != nil {
					toolResultBlock = types.ToolResultBlock{
						Content: []types.ToolResultContentBlock{
							&types.ToolResultContentBlockMemberJson{
								Value: document.NewLazyDocument(map[string]any{
									"error": toolErr.Error(),
								}),
							},
						},
						ToolUseId: tool.Value.ToolUseId,
						Status:    types.ToolResultStatusError,
						Type:      nil,
					}
				} else {
					toolResultBlock = types.ToolResultBlock{
						Content: []types.ToolResultContentBlock{
							&types.ToolResultContentBlockMemberJson{
								Value: document.NewLazyDocument(toolResult),
							},
						},
						ToolUseId: tool.Value.ToolUseId,
						Status:    types.ToolResultStatusSuccess,
						Type:      nil,
					}
				}
				messages = append(messages, types.Message{
					Role: types.ConversationRoleUser,
					Content: []types.ContentBlock{
						&types.ContentBlockMemberToolResult{
							Value: toolResultBlock,
						},
					},
				})
			}
		}

		log.Println(resp.StopReason)
		if resp.StopReason == types.StopReasonEndTurn {
			break
		}
	}
}

type PropertyDesc struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	required    bool
}

func stringProp(desc string) PropertyDesc {
	return PropertyDesc{
		Type:        "string",
		Description: desc,
	}
}

func requiredStringProp(desc string) PropertyDesc {
	prop := stringProp(desc)
	prop.required = true
	return prop
}

type ToolSchema struct {
	Type       string                  `json:"type"`
	Properties map[string]PropertyDesc `json:"properties"`
	Required   []string                `json:"required"`
}

type toolDesc map[string]PropertyDesc

func marshalToolDesc(properties toolDesc) (j map[string]any) {
	required := []string{}
	for propName, prop := range properties {
		if prop.required {
			required = append(required, propName)
		}
	}
	schema := ToolSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
	jb, _ := json.Marshal(schema)
	_ = json.Unmarshal(jb, &j)
	return
}

func toolSpec(name string, desc string, params toolDesc) *types.ToolMemberToolSpec {
	return &types.ToolMemberToolSpec{
		Value: types.ToolSpecification{
			Name:        aws.String(name),
			Description: aws.String(desc),
			InputSchema: &types.ToolInputSchemaMemberJson{
				Value: document.NewLazyDocument(marshalToolDesc(params)),
			},
		},
	}
}
