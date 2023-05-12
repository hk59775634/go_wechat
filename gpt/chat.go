package gpt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/qingconglaixueit/wechatbot/config"
)

type ChatCompletionChoice struct {
	Message ChatCompletionMessage `json:"message"`
	FinishReason string           `json:"finish_reason"`
	Index        int              `json:"index"`
}

type ChatGPTResponseBody struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
	Error   struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
		Code    interface{} `json:"code"`
	} `json:"error"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name string `json:"name,omitempty"`
}

type ChatCompletionRequest struct {
	Model            string                  `json:"model"`
	Messages         []ChatCompletionMessage `json:"messages"`
	MaxTokens        int                     `json:"max_tokens,omitempty"`
	Temperature      float32                 `json:"temperature,omitempty"`
	TopP             float32                 `json:"top_p,omitempty"`
	N                int                     `json:"n,omitempty"`
	Stream           bool                    `json:"stream,omitempty"`
	Stop             []string                `json:"stop,omitempty"`
	PresencePenalty  float32                 `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32                 `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int          `json:"logit_bias,omitempty"`
	User             string                  `json:"user,omitempty"`
}

func ChatCompletions(msg string) (string, error) {
	var gptResponseBody *ChatGPTResponseBody
	var resErr error
	for retry := 1; retry <= 3; retry++ {
		if retry > 1 {
			time.Sleep(time.Duration(retry-1) * 100 * time.Millisecond)
		}
		gptResponseBody, resErr = httpRequestChatCompletions(msg, retry)
		if resErr != nil {
			log.Printf("gpt request(%d) error: %v\n", retry, resErr)
			continue
		}
		if gptResponseBody.Error.Message == "" {
			break
		}
	}
	if resErr != nil {
		return "", resErr
	}
	// log.Printf("gpt response: %+v\n", gptResponseBody)

	var reply string
	if gptResponseBody != nil && len(gptResponseBody.Choices) > 0 {
		reply = gptResponseBody.Choices[0].Message.Content
		// log.Printf("gpt response: %+v\n", reply)
	}
	return reply, nil
}


func httpRequestChatCompletions(msg string, runtimes int) (*ChatGPTResponseBody, error) {
	cfg := config.LoadConfig()
	if cfg.ApiUrl == "" {
		return nil, errors.New("api url required")
	}
	if cfg.ApiKey == "" {
		return nil, errors.New("api key required")
	}

	requestBody := ChatCompletionRequest{
		Model:      cfg.Model,
		Messages:   []ChatCompletionMessage{
			{
				Role:     "user",
				Content:  msg,
			},
		},
		Stream:     false,
	}

	requestData, err := json.MarshalIndent(requestBody, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("json.Marshal requestBody error: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, cfg.ApiUrl + "/v1/chat/completions", bytes.NewBuffer(requestData))
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest error: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)
	client := &http.Client{Timeout: 60 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.Do error: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll error: %v", err)
	}

	// log.Printf("gpt response(%d) json: %s\n", runtimes, string(body))

	gptResponseBody := &ChatGPTResponseBody{}
	err = json.Unmarshal(body, gptResponseBody)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal responseBody error: %v", err)
	}
	return gptResponseBody, nil
}
