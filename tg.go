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
	"reflect"
	"regexp"
	"slices"
	"time"
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
	"",
}

var ErrUnknownParseMode = errors.New("unknown parse_mode")

func (m ParseMode) Validate() error {
	if !slices.Contains(parseModeList, m) {
		return ErrUnknownParseMode
	}

	return nil
}

type BaseMessage struct {
	ChatID    int64     `json:"chat_id"`
	Text      string    `json:"text"`
	ParseMode ParseMode `json:"parse_mode,omitempty"`
}

const MaxTextSize int = 4096

var (
	ErrEmptyChatID = errors.New("empty chat_id")
	ErrEmptyText   = errors.New("empty text")
	ErrTextTooLong = errors.New("text too long")
)

func (bm *BaseMessage) Validate() error {
	if bm.ChatID == 0 {
		return ErrEmptyChatID
	}

	if bm.Text == "" {
		return ErrEmptyText
	}

	if len(bm.Text) > MaxTextSize {
		return ErrTextTooLong
	}

	if err := bm.ParseMode.Validate(); err != nil {
		return err
	}

	return nil
}

type SendMessage struct {
	BaseMessage
	MessageThreadID       int64 `json:"message_thread_id,omitempty"`
	DisableWebPagePreview bool  `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool  `json:"disable_notification,omitempty"`
	ProtectContent        bool  `json:"protect_content,omitempty"`
}

var ErrIncorrectMessageThreadID = errors.New("incorrect message_thread_id")

func (sm *SendMessage) Validate() error {
	if err := sm.BaseMessage.Validate(); err != nil {
		return err
	}

	if sm.MessageThreadID < 0 {
		return ErrIncorrectMessageThreadID
	}

	return nil
}

type SendOption func(*SendMessage)

func NewSendMessage(chatID int64, text string, opts ...SendOption) (*SendMessage, error) {
	sm := new(SendMessage)

	for _, opt := range opts {
		opt(sm)
	}

	sm.ChatID = chatID
	sm.Text = text

	if err := sm.Validate(); err != nil {
		return nil, fmt.Errorf("SendMessage: %w", err)
	}

	return sm, nil
}

func ParseModeSendOption(mode ParseMode) SendOption {
	return func(sm *SendMessage) {
		sm.ParseMode = mode
	}
}

func MessageThreadIDSendOption(threadID int64) SendOption {
	return func(sm *SendMessage) {
		sm.MessageThreadID = threadID
	}
}

func DisableWebPagePreviewSendOption(disable bool) SendOption {
	return func(sm *SendMessage) {
		sm.DisableWebPagePreview = disable
	}
}

func DisableNotificationSendOption(disable bool) SendOption {
	return func(sm *SendMessage) {
		sm.DisableNotification = disable
	}
}

func ProtectContentSendOption(protect bool) SendOption {
	return func(sm *SendMessage) {
		sm.ProtectContent = protect
	}
}

type EditMessage struct {
	MessageID int64 `json:"message_id"`
	BaseMessage
}

var ErrIncorrectMessageID = errors.New("incorrect message_id")

func (em *EditMessage) Validate() error {
	if em.MessageID <= 0 {
		return ErrIncorrectMessageID
	}

	return em.BaseMessage.Validate()
}

type EditOption func(*EditMessage)

func NewEditMessage(chatID int64, messageID int64, text string, opts ...EditOption) (*EditMessage, error) {
	em := new(EditMessage)

	for _, opt := range opts {
		opt(em)
	}

	em.ChatID = chatID
	em.MessageID = messageID
	em.Text = text

	if err := em.Validate(); err != nil {
		return nil, fmt.Errorf("EditMessage: %w", err)
	}

	return em, nil
}

func ParseModeEditOption(mode ParseMode) EditOption {
	return func(em *EditMessage) {
		em.ParseMode = mode
	}
}

type DeleteMessage struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

func (dm *DeleteMessage) Validate() error {
	if dm.ChatID == 0 {
		return ErrEmptyChatID
	}

	if dm.MessageID <= 0 {
		return ErrIncorrectMessageID
	}

	return nil
}

func NewDeleteMessage(chatID int64, messageID int64) (*DeleteMessage, error) {
	dm := new(DeleteMessage)

	dm.ChatID = chatID
	dm.MessageID = messageID

	if err := dm.Validate(); err != nil {
		return nil, fmt.Errorf("DeleteMessage: %w", err)
	}

	return dm, nil
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	UserName  string `json:"username,omitempty"`
}

type Message struct {
	MessageID int64 `json:"message_id"`
	Date      int   `json:"date"`
}

type TG interface {
	GetMe(ctx context.Context) (*User, error)
	SendMessage(ctx context.Context, chatID int64, text string, opts ...SendOption) (*Message, error)
	EditMessage(ctx context.Context, chatID, messageID int64, text string, opts ...EditOption) (*Message, error)
	DeleteMessage(ctx context.Context, chatID, messageID int64) (bool, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	http     HTTPClient
	endpoint string
}

var _ TG = (*Client)(nil)

type Option func(*Client) error

var (
	ErrIncorrectScheme = errors.New("incorrect scheme")
	ErrEmptyHost       = errors.New("empty host")
)

func WithAPIServer(server string) Option {
	return func(cl *Client) error {
		url, err := url.ParseRequestURI(server)
		if err != nil {
			return fmt.Errorf("apiserver: %w", err)
		}

		if url.Scheme != "http" && url.Scheme != "https" {
			return fmt.Errorf("apiserver: url: %w", ErrIncorrectScheme)
		}

		if url.Host == "" {
			return fmt.Errorf("apiserver: url: %w", ErrEmptyHost)
		}

		cl.endpoint = server

		return nil
	}
}

var ErrHTTPClientNil = errors.New("httpclient is nil")

func WithHTTPClient(client HTTPClient) Option {
	return func(cl *Client) error {
		if client == nil {
			return ErrHTTPClientNil
		}

		cl.http = client

		return nil
	}
}

const defaultAPIServer = "https://api.telegram.org"

//nolint:gomnd,gochecknoglobals
var defaultHTTPClient = &http.Client{
	Timeout: 2 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 10 * time.Second,
	},
}

var regexpToken = regexp.MustCompile(`^([\d]+):([\d\w\-]+)$`)

var ErrIncorrentToken = errors.New("incorrect token")

func NewClient(token string, options ...Option) (*Client, error) {
	if !regexpToken.MatchString(token) {
		return nil, fmt.Errorf("Client: %w", ErrIncorrentToken)
	}

	client := new(Client)
	client.endpoint = defaultAPIServer

	for _, opt := range options {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("Client: %w", err)
		}
	}

	if client.http == nil {
		client.http = defaultHTTPClient
	}

	client.endpoint += "/bot" + token + "/"

	return client, nil
}

type Response struct {
	Result interface{} `json:"result,omitempty"`
	ResponseError
}

type ResponseError struct {
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  struct {
		RetryAfter int `json:"retry_after,omitempty"`
	} `json:"parameters,omitempty"`
}

func (r ResponseError) Error() string {
	return r.Description
}

var (
	ErrValueNil             = errors.New("value is nil")
	ErrValueNotPtr          = errors.New("value not ptr")
	ErrValueNotStructOrBool = errors.New("value not struct or bool")
)

func validate(v any) error {
	if v == nil {
		return ErrValueNil
	}

	value := reflect.ValueOf(v)

	if value.Type().Kind() != reflect.Ptr {
		return ErrValueNotPtr
	}

	if value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct && value.Kind() != reflect.Bool {
		return ErrValueNotStructOrBool
	}

	return nil
}

func (c *Client) API(ctx context.Context, method string, req, resp any) error {
	var reqBody io.Reader

	if req != nil {
		if err := validate(req); err != nil {
			return fmt.Errorf("validate: req %w", err)
		}

		body, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("request: json: %w", err)
		}

		reqBody = bytes.NewReader(body)
	}

	if err := validate(resp); err != nil {
		return fmt.Errorf("validate: resp %w", err)
	}

	url := c.endpoint + method

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reqBody)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}

	httpReq.Header.Add("Content-Type", "application/json")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}

	defer httpResp.Body.Close()

	respBody := new(Response)
	respBody.Result = resp

	if err := json.NewDecoder(httpResp.Body).Decode(respBody); err != nil {
		return fmt.Errorf("response: json: %w", err)
	}

	if !respBody.Ok {
		return fmt.Errorf("response: %w", respBody.ResponseError)
	}

	return nil
}

const getMeMethod = "getMe"

func (c *Client) GetMe(ctx context.Context) (*User, error) {
	resp := new(User)

	if err := c.API(ctx, getMeMethod, nil, resp); err != nil {
		return nil, fmt.Errorf("GetMe: %w", err)
	}

	return resp, nil
}

const sendMessageMethod = "sendMessage"

func (c *Client) SendMessage(ctx context.Context,
	chatID int64, text string, opts ...SendOption,
) (*Message, error) {
	req, err := NewSendMessage(chatID, text, opts...)
	if err != nil {
		return nil, fmt.Errorf("SendMessage: %w", err)
	}

	resp := new(Message)

	if err := c.API(ctx, sendMessageMethod, req, resp); err != nil {
		return nil, fmt.Errorf("SendMessage: %w", err)
	}

	return resp, nil
}

const editMessageTextMethod = "editMessageText"

func (c *Client) EditMessage(ctx context.Context,
	chatID, messageID int64, text string, opts ...EditOption,
) (*Message, error) {
	req, err := NewEditMessage(chatID, messageID, text, opts...)
	if err != nil {
		return nil, fmt.Errorf("EditMessage: %w", err)
	}

	resp := new(Message)

	if err := c.API(ctx, editMessageTextMethod, req, resp); err != nil {
		return nil, fmt.Errorf("EditMessage: %w", err)
	}

	return resp, nil
}

const deleteMessageMethod = "deleteMessage"

func (c *Client) DeleteMessage(ctx context.Context, chatID, messageID int64) (bool, error) {
	req, err := NewDeleteMessage(chatID, messageID)
	if err != nil {
		return false, fmt.Errorf("DeleteMessage: %w", err)
	}

	resp := false

	if err := c.API(ctx, deleteMessageMethod, req, &resp); err != nil {
		return false, fmt.Errorf("DeleteMessage: %w", err)
	}

	return resp, nil
}
