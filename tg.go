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
	"regexp"
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

var parseModeList = []ParseMode{ //nolint:gochecknoglobals
	MarkdownV2ParseMode,
	MarkdownParseMode,
	HTMLParseMode,
}

var (
	ErrInvalidScheme   = errors.New("invalid scheme")
	ErrEmptyHost       = errors.New("empty host")
	ErrClientNil       = errors.New("client is nil")
	ErrModeUnknown     = errors.New("unknown mode")
	ErrInvalidToken    = errors.New("invalid token")
	ErrInvalidThreadID = errors.New("invalid thread id")
	ErrEmptyText       = errors.New("empty text")
	ErrExceedsMaxText  = errors.New("exeeds max text")
)

// User .
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	UserName  string `json:"username,omitempty"`
}

// Chat .
type Chat struct {
	ChatID                int64  `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	MessageThreadID       int64  `json:"message_thread_id,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool   `json:"disable_notification,omitempty"`
	ProtectContent        bool   `json:"protect_content,omitempty"`
}

type ChatOption func(*Chat) error

func ChatParseMode(mode ParseMode) ChatOption {
	return func(ch *Chat) error {
		for _, item := range parseModeList {
			if mode == item {
				ch.ParseMode = string(mode)

				return nil
			}
		}

		return fmt.Errorf("MessageParseMode: %w", ErrModeUnknown)
	}
}

func ChatMessageThreadID(threadID int64) ChatOption {
	return func(ch *Chat) error {
		if threadID < 0 {
			return ErrInvalidThreadID
		}

		ch.MessageThreadID = threadID

		return nil
	}
}

func ChatDisableWebPagePreview(disable bool) ChatOption {
	return func(ch *Chat) error {
		ch.DisableWebPagePreview = disable

		return nil
	}
}

func ChatDisableNotification(disable bool) ChatOption {
	return func(ch *Chat) error {
		ch.DisableNotification = disable

		return nil
	}
}

func ChatProtectContent(protect bool) ChatOption {
	return func(ch *Chat) error {
		ch.ProtectContent = protect

		return nil
	}
}

// Message .
type Message struct {
	MessageID int `json:"message_id"`
	Date      int `json:"date"`
}

// APIResponse .
type APIResponse struct {
	Result interface{} `json:"result,omitempty"`
	APIResponseError
}

// APIResponseError .
type APIResponseError struct {
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  struct {
		RetryAfter int `json:"retry_after,omitempty"`
	} `json:"parameters,omitempty"`
}

func (r APIResponseError) Error() string {
	return r.Description
}

var (
	regexpBotToken = regexp.MustCompile(`/bot([\d]+):([\d\w\-]+)/`)
	regexpToken    = regexp.MustCompile(`^([\d]+):([\d\w\-]+)$`)
)

type redactError struct {
	err error
}

func newRedactError(err error) error {
	return &redactError{
		err: err,
	}
}

func (e *redactError) Error() string {
	return regexpBotToken.ReplaceAllString(e.err.Error(), "/bot*****/")
}

func (e *redactError) Unwrap() error {
	return e.err
}

type Option func(*TG) error

func APIServer(server string) Option {
	return func(tg *TG) error {
		url, err := url.ParseRequestURI(server)
		if err != nil {
			return fmt.Errorf("APIServer: %w", err)
		}

		if url.Scheme != "http" && url.Scheme != "https" {
			return fmt.Errorf("APIServer: %w", ErrInvalidScheme)
		}

		if url.Host == "" {
			return fmt.Errorf("APIServer: %w", ErrEmptyHost)
		}

		tg.endpoint = server

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

type tg interface {
	GetMe(ctx context.Context) (*User, error)
	SendMessage(ctx context.Context, chatID int64, text string, opts ...ChatOption) (*Message, error)
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// TG .
type TG struct {
	http     httpClient
	endpoint string
}

var _ tg = (*TG)(nil)

//nolint:gochecknoglobals,gomnd
var defaultHTTPClient = &http.Client{
	Timeout: 2 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 10 * time.Second,
	},
}

// NewTG .
func NewTG(token string, options ...Option) (*TG, error) {
	if !regexpToken.MatchString(token) {
		return nil, fmt.Errorf("TG: %w", ErrInvalidToken)
	}

	tg := &TG{
		http:     nil,
		endpoint: apiServer,
	}

	for _, opt := range options {
		if err := opt(tg); err != nil {
			return nil, fmt.Errorf("TG: %w", err)
		}
	}

	if tg.http == nil {
		tg.http = defaultHTTPClient
	}

	tg.endpoint += "/bot" + token + "/"

	return tg, nil
}

func (t *TG) makeMessage(chatID int64, text string, opts ...ChatOption) (io.Reader, error) {
	chat := &Chat{
		ChatID: chatID,
		Text:   text,
	}

	for _, opt := range opts {
		if err := opt(chat); err != nil {
			return nil, fmt.Errorf("makeMessage: option: %w", err)
		}
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

// GetMe .
func (t *TG) GetMe(ctx context.Context) (*User, error) {
	req, err := t.makeRequest(ctx, apiMethodGetMe, nil)
	if err != nil {
		return nil, fmt.Errorf("GetMe: %w", err)
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GetMe: http: %w", newRedactError(err))
	}

	user := new(User)

	if err := t.makeResponse(resp, user); err != nil {
		return nil, fmt.Errorf("GetMe: %w", err)
	}

	return user, nil
}

const MaxTextSize int = 4096

// SendMessage .
func (t *TG) SendMessage(ctx context.Context, chatID int64, text string, opts ...ChatOption) (*Message, error) {
	if text == "" {
		return nil, fmt.Errorf("SendMessage: %w", ErrEmptyText)
	}

	if len(text) > MaxTextSize {
		return nil, fmt.Errorf("SendMessage: %w", ErrExceedsMaxText)
	}

	reader, err := t.makeMessage(chatID, text, opts...)
	if err != nil {
		return nil, fmt.Errorf("SendMessage: %w", err)
	}

	req, err := t.makeRequest(ctx, apiMethodSendMessage, reader)
	if err != nil {
		return nil, fmt.Errorf("SendMessage: %w", err)
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SendMessage: http: %w", newRedactError(err))
	}

	message := new(Message)

	if err := t.makeResponse(resp, message); err != nil {
		return nil, fmt.Errorf("SendMessage: %w", err)
	}

	return message, nil
}
