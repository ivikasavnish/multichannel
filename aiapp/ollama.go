package ollama

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// Client manages communication with the Ollama API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Ollama API client.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

// APIError represents an error response from the API.
type APIError struct {
	Message string `json:"message"`
}

// GenerateRequest represents a request to the /generate endpoint.
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream,omitempty"`
	Format string `json:"format,omitempty"`
	//Options map[string]any `json:"options,omitempty"`
}

// GenerateResponse represents a response from the /generate endpoint.
type GenerateResponse struct {
	Model         string        `json:"model"`
	CreatedAt     string        `json:"created_at"`
	Response      string        `json:"response"`
	Done          bool          `json:"done"`
	TotalDuration time.Duration `json:"total_duration"`
}

// GenerateCompletion sends a prompt to the /generate endpoint and retrieves a response.
func (c *Client) GenerateCompletion(req *GenerateRequest) (*GenerateResponse, error) {
	url := c.BaseURL + "/api/generate"
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		apiErr := &APIError{}
		if err := json.NewDecoder(httpResp.Body).Decode(apiErr); err != nil {
			return nil, errors.New("failed to decode error response")
		}
		return nil, errors.New(apiErr.Message)
	}

	resp := &GenerateResponse{}
	if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// ChatRequest represents a request to the /chat endpoint.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
}

// GenerateChat handles chat-like conversation with the /chat endpoint.
func (c *Client) GenerateChat(req *ChatRequest) (*ChatResponse, error) {
	url := c.BaseURL + "/api/chat"
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		apiErr := &APIError{}
		if err := json.NewDecoder(httpResp.Body).Decode(apiErr); err != nil {
			return nil, errors.New("failed to decode error response")
		}
		return nil, errors.New(apiErr.Message)
	}

	resp := &ChatResponse{}
	if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// ListModelsResponse represents the response from the /tags endpoint.
type ListModelsResponse struct {
	Models []struct {
		Name       string `json:"name"`
		ModifiedAt string `json:"modified_at"`
		Size       int64  `json:"size"`
	} `json:"models"`
}

// ListModels retrieves the list of available models.
func (c *Client) ListModels() (*ListModelsResponse, error) {
	url := c.BaseURL + "/api/tags"
	httpResp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		apiErr := &APIError{}
		if err := json.NewDecoder(httpResp.Body).Decode(apiErr); err != nil {
			return nil, errors.New("failed to decode error response")
		}
		return nil, errors.New(apiErr.Message)
	}

	resp := &ListModelsResponse{}
	if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// ShowModelRequest represents a request to the /show endpoint.
type ShowModelRequest struct {
	Model string `json:"model"`
}

type ShowModelResponse struct {
	Details struct {
		ParentModel string `json:"parent_model"`
		Format      string `json:"format"`
		Family      string `json:"family"`
	} `json:"details"`
}

// ShowModel retrieves details about a specific model.
func (c *Client) ShowModel(req *ShowModelRequest) (*ShowModelResponse, error) {
	url := c.BaseURL + "/api/show"
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		apiErr := &APIError{}
		if err := json.NewDecoder(httpResp.Body).Decode(apiErr); err != nil {
			return nil, errors.New("failed to decode error response")
		}
		return nil, errors.New(apiErr.Message)
	}

	resp := &ShowModelResponse{}
	if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
		return nil, err
	}

	return resp, nil
}
