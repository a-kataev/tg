package tg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const apiServer = "https://api.telegram.org"

type apiMethod string

const (
	apiMethodGetMe       apiMethod = "getMe"
	apiMethodSendMessage apiMethod = "sendMessage"
)

type ParseMode string

const (
	MarkdownV2ParseMode = "MarkdownV2"
	MarkdownParseMode   = "Markdown"
	HTMLParseMode       = "HTML"
)

var parseModeList = []ParseMode{ //nolint
	MarkdownV2ParseMode,
	MarkdownParseMode,
	HTMLParseMode,
}

var (
	ErrInvalidScheme = errors.New("invalid scheme")
	ErrEmptyHost     = errors.New("empty host")
	ErrClientNil     = errors.New("client is nil")
	ErrModeUnknown   = errors.New("unknown mode")
)

// User -
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"` //nolint
	UserName  string `json:"username,omitempty"`
}

// Chat -
type Chat struct {
	ChatID    int64  `json:"chat_id,omitempty"` //nolint
	Text      string `json:"text,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"` //nolint
}

// Message -
type Message struct {
	MessageID int `json:"message_id"` //nolint
	Date      int `json:"date"`
}

// APIResponse -
type APIResponse struct {
	Result interface{} `json:"result,omitempty"`
	APIResponseError
}

// APIResponseError -
type APIResponseError struct {
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code,omitempty"` //nolint
	Description string `json:"description,omitempty"`
	Parameters  struct {
		RetryAfter int `json:"retry_after,omitempty"` //nolint
	} `json:"parameters,omitempty"`
}

func (r APIResponseError) Error() string {
	return r.Description
}

type Option func(t *TG) error

func APIServer(server string) Option {
	return func(t *TG) error {
		u, err := url.ParseRequestURI(server)
		if err != nil {
			return fmt.Errorf("APIServer: %w", err)
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("APIServer: %w", ErrInvalidScheme)
		}

		if u.Host == "" {
			return fmt.Errorf("APIServer: %w", ErrEmptyHost)
		}

		t.endpoint = server

		return nil
	}
}

func HTTPClient(client *http.Client) Option {
	return func(t *TG) error {
		if client == nil {
			return fmt.Errorf("HTTPClient: %w", ErrClientNil)
		}

		t.http = client

		return nil
	}
}

func MessageParseMode(mode ParseMode) Option {
	return func(t *TG) error {
		for _, m := range parseModeList {
			if mode == m {
				t.parseMode = string(mode)

				return nil
			}
		}

		return fmt.Errorf("MessageParseMode: %w", ErrModeUnknown)
	}
}

type tg interface {
	GetMe(ctx context.Context) (*User, error)
	SendMessage(ctx context.Context, chatID int64, text string) (*Message, error)
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// TG -
type TG struct {
	http      httpClient
	endpoint  string
	parseMode string
}

var _ tg = (*TG)(nil)

// NewTG -
func NewTG(token string, options ...Option) (*TG, error) {
	t := &TG{
		http:      nil,
		endpoint:  apiServer,
		parseMode: string(MarkdownParseMode),
	}

	for _, opt := range options {
		if err := opt(t); err != nil {
			return nil, err
		}
	}

	if t.http == nil {
		client := http.DefaultClient
		client.Timeout = 2 * time.Second    //nolint
		client.Transport = &http.Transport{ //nolint
			MaxIdleConns:    10,               //nolint
			IdleConnTimeout: 10 * time.Second, //nolint
		}

		t.http = client
	}

	t.endpoint += "/bot" + token + "/"

	return t, nil
}

func (t *TG) makeMessage(chatID int64, text string) (io.Reader, error) {
	chat := &Chat{
		ChatID:    chatID,
		Text:      text,
		ParseMode: t.parseMode,
	}

	body, err := json.Marshal(chat)
	if err != nil {
		return nil, fmt.Errorf("makeMessage: json: %w", err)
	}

	return bytes.NewReader(body), nil
}

func (t *TG) makeRequest(ctx context.Context, method apiMethod, reader io.Reader) (*http.Request, error) {
	url := t.endpoint + string(method)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reader)
	if err != nil {
		return nil, fmt.Errorf("makeMessage: request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func (t *TG) makeResponse(resp *http.Response, result interface{}) error {
	apiResp := new(APIResponse)
	apiResp.Result = result

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(apiResp); err != nil {
		return fmt.Errorf("makeResponse: body: %w", err)
	}

	if !apiResp.Ok {
		return apiResp.APIResponseError
	}

	return nil
}

// GetMe -
func (t *TG) GetMe(ctx context.Context) (*User, error) {
	req, err := t.makeRequest(ctx, apiMethodGetMe, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GetMe: http: %w", err)
	}

	user := new(User)

	if err := t.makeResponse(resp, user); err != nil {
		return nil, err
	}

	return user, nil
}

// SendMessage -
func (t *TG) SendMessage(ctx context.Context, chatID int64, text string) (*Message, error) {
	reader, err := t.makeMessage(chatID, text)
	if err != nil {
		return nil, err
	}

	req, err := t.makeRequest(ctx, apiMethodSendMessage, reader)
	if err != nil {
		return nil, err
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SendMessage: http: %w", err)
	}

	message := new(Message)

	if err := t.makeResponse(resp, message); err != nil {
		return nil, err
	}

	return message, nil
}
