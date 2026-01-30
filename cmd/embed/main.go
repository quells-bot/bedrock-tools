package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

const model = "amazon.titan-embed-text-v2:0"

func main() {
	ctx := context.Background()
	conf, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Printf("failed to load aws config: %v", err)
		return
	}

	bd := bedrockruntime.NewFromConfig(conf)

	resp, err := bd.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(model),
		Accept:      aws.String("application/json"),
		Body:        embed("hello world"),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		log.Printf("failed to invoke model: %v", err)
		return
	}

	var embedResp EmbedResponse
	_ = json.Unmarshal(resp.Body, &embedResp)
	log.Println(embedResp)
}

type EmbedRequest struct {
	InputText        string   `json:"inputText"`
	OutputDimensions int      `json:"dimensions"`
	Normalize        bool     `json:"normalize"`
	Types            []string `json:"embeddingTypes"` // float and/or binary
}

func embed(text string) []byte {
	req := EmbedRequest{
		InputText:        text,
		OutputDimensions: 1024,
		Normalize:        true,
		Types:            []string{"float"},
	}
	j, _ := json.Marshal(req)
	return j
}

type EmbedResponse struct {
	Embedding       []float32 `json:"embedding"`
	InputTokenCount int       `json:"inputTextTokenCount"`
}
