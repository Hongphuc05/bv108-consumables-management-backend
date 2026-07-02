package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultGeminiTimeout = 60 * time.Second

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

type GeminiProxyService struct {
	apiKey          string
	model           string
	enableWebSearch bool
	maxOutputTokens int
	httpClient      *http.Client
}

func NewGeminiProxyService(apiKey, model string, enableWebSearch bool, maxOutputTokens int) *GeminiProxyService {
	return &GeminiProxyService{
		apiKey:          strings.TrimSpace(apiKey),
		model:           strings.TrimSpace(model),
		enableWebSearch: enableWebSearch,
		maxOutputTokens: maxOutputTokens,
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

	payload := map[string]any{
		"contents": req.Contents,
		"generationConfig": map[string]any{
			"temperature":     0.4,
			"maxOutputTokens": s.maxOutputTokens,
		},
	}

	if s.enableWebSearch {
		payload["tools"] = []map[string]any{{"google_search": map[string]any{}}}
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		s.model,
	)

	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(rawPayload))
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}

	var parsed GeminiProxyResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("Gemini returned invalid JSON")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := ""
		if parsed.Error != nil {
			message = strings.TrimSpace(parsed.Error.Message)
		}
		if message == "" {
			message = fmt.Sprintf("Gemini API lỗi (%d)", resp.StatusCode)
		}
		return nil, resp.StatusCode, fmt.Errorf("%s", message)
	}

	return &parsed, http.StatusOK, nil
}
