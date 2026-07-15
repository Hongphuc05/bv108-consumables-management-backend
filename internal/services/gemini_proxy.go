package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultGeminiAPIBaseURL  = "https://generativelanguage.googleapis.com/v1beta"
	defaultGeminiTimeout     = 60 * time.Second
	defaultGeminiTemperature = 0.4
)

type GeminiTextPart struct {
	Text string `json:"text"`
}

type GeminiContent struct {
	Role  string           `json:"role"`
	Parts []GeminiTextPart `json:"parts"`
}

type GeminiProxyRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiProxyResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		GroundingMetadata struct {
			GroundingChunks []struct {
				Web struct {
					URI   string `json:"uri"`
					Title string `json:"title"`
				} `json:"web"`
			} `json:"groundingChunks"`
		} `json:"groundingMetadata"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type GeminiProxyConfig struct {
	APIKey          string
	Model           string
	APIBaseURL      string
	EnableWebSearch bool
	MaxOutputTokens int
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type geminiGenerateRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
	Tools            []map[string]any       `json:"tools,omitempty"`
}

type GeminiProxyService struct {
	apiKey          string
	model           string
	apiBaseURL      string
	enableWebSearch bool
	maxOutputTokens int
	httpClient      *http.Client
}

func NewGeminiProxyService(cfg GeminiProxyConfig) *GeminiProxyService {
	apiBaseURL := strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = defaultGeminiAPIBaseURL
	}

	return &GeminiProxyService{
		apiKey:          strings.TrimSpace(cfg.APIKey),
		model:           strings.TrimSpace(cfg.Model),
		apiBaseURL:      apiBaseURL,
		enableWebSearch: cfg.EnableWebSearch,
		maxOutputTokens: cfg.MaxOutputTokens,
		httpClient: &http.Client{
			Timeout: defaultGeminiTimeout,
		},
	}
}

func (s *GeminiProxyService) IsConfigured() bool {
	return s.apiKey != "" && s.model != ""
}

func (s *GeminiProxyService) GenerateContent(req GeminiProxyRequest) (*GeminiProxyResponse, int, error) {
	if !s.IsConfigured() {
		return nil, http.StatusServiceUnavailable, fmt.Errorf("Gemini backend is not configured")
	}

	rawPayload, err := json.Marshal(s.buildGenerateRequest(req))
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, s.generateContentURL(), bytes.NewReader(rawPayload))
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	defer resp.Body.Close()

	parsed, err := decodeGeminiResponse(resp.Body)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, resp.StatusCode, geminiResponseError(parsed, resp.StatusCode)
	}

	return parsed, http.StatusOK, nil
}

func (s *GeminiProxyService) buildGenerateRequest(req GeminiProxyRequest) geminiGenerateRequest {
	payload := geminiGenerateRequest{
		Contents: req.Contents,
		GenerationConfig: geminiGenerationConfig{
			Temperature:     defaultGeminiTemperature,
			MaxOutputTokens: s.maxOutputTokens,
		},
	}

	if s.enableWebSearch {
		payload.Tools = []map[string]any{{"google_search": map[string]any{}}}
	}

	return payload
}

func (s *GeminiProxyService) generateContentURL() string {
	return fmt.Sprintf("%s/models/%s:generateContent", s.apiBaseURL, url.PathEscape(s.model))
}

func decodeGeminiResponse(body io.Reader) (*GeminiProxyResponse, error) {
	var parsed GeminiProxyResponse
	if err := json.NewDecoder(body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("Gemini returned invalid JSON")
	}
	return &parsed, nil
}

func geminiResponseError(resp *GeminiProxyResponse, statusCode int) error {
	if resp.Error != nil {
		if message := strings.TrimSpace(resp.Error.Message); message != "" {
			return fmt.Errorf("%s", message)
		}
	}
	return fmt.Errorf("Gemini API lỗi (%d)", statusCode)
}
