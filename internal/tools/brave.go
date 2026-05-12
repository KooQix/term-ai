package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/KooQix/term-ai/internal/config"
)

const WebSearchType ToolType = "web_search"

type BraveResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type braveResponse struct {
	Web struct {
		Results []BraveResult `json:"results"`
	} `json:"web"`
}

/// Tool implementation for Brave Search API - matching the tools.Tool interface

type braveSearchConfig struct {
	ApiKey string `yaml:"api_key" sensitive:"true"`
}
type braveSearch struct {
	config braveSearchConfig
}

func (b *braveSearch) Name() string {
	return string(WebSearchType)
}

func (b *braveSearch) Execute(name, argsJSON string) (string, error) {
	var args struct {
		Query string `json:"query"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}

	if args.Count <= 0 {
		args.Count = 5
	} else if args.Count > 10 {
		args.Count = 10
	}

	endpoint := fmt.Sprintf(
		"https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(args.Query), args.Count,
	)

	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Set("X-Subscription-Token", b.config.ApiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("brave %d: %s", resp.StatusCode, b)
	}

	var br braveResponse
	if err := json.NewDecoder(resp.Body).Decode(&br); err != nil {
		return "", err
	}
	results, err := json.Marshal(br.Web.Results)
	if err != nil {
		return "", err
	}
	return string(results), nil
}

func (b *braveSearch) Tool() Tool {
	return Tool{
		Type: "function",
		Function: Function{
			Name:        WebSearchType,
			Description: "Search the web via Brave. Use when you need current/factual info beyond your training data.",

			Parameters: FunctionParameters{
				Type: "object",
				Properties: map[string]FunctionParameter{
					"query": {
						Type:        "string",
						Description: "Search query",
					},
					"count": {
						Type:        "integer",
						Description: "Number of results (1-10, default 5)",
					},
				},
				Required: []string{"query"},
			},
		},
	}
}

// Register the tool on package initialization
func init() {
	tool := &braveSearch{}

	// First, register the tool with default config (empty API key)
	config.RegisterToolConfig(tool.Name(), braveSearchConfig{})

	// Then attempt to register it
	var cf braveSearchConfig

	err := config.ParseToolConfig(tool.Name(), &cf)
	if err != nil {
		// Failed to parse config, skip registering the tool
		return
	}

	if cf.ApiKey == "" {
		// API key is required, skip registering the tool if it's not set
		return
	}

	tool.config = cf

	// Make it usable by the provider
	registerTool(tool)
}
