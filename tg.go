package tg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var apiServer = "https://api.telegram.org" //nolint

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"` //nolint
	UserName  string `json:"username,omitempty"`
}

type Chat struct {
	ChatID    int64  `json:"chat_id,omitempty"` //nolint
	Text      string `json:"text,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"` //nolint
}

type Message struct {
	MessageID int `json:"message_id"` //nolint
	Date      int `json:"date"`
}

type APIResponse struct {
	Result interface{} `json:"result,omitempty"`
	APIResponseError
}

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

type apiMethod string

const (
	apiMethodGetMe       apiMethod = "getMe"
	apiMethodSendMessage apiMethod = "sendMessage"
)

type tg interface {
	GetMe(ctx context.Context) (*User, error)
	SendMessage(ctx context.Context, chatID int64, text string) (*Message, error)
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

var _ tg = (*TG)(nil)

func NewTG(token string) *TG {
	client := http.DefaultClient
	client.Timeout = 2 * time.Second    //nolint
	client.Transport = &http.Transport{ //nolint
		MaxIdleConns:    10,               //nolint
		IdleConnTimeout: 10 * time.Second, //nolint
	}

	return NewTGWithClient(token, client)
}

func NewTGWithClient(token string, client *http.Client) *TG {
	return &TG{
		http:     client,
		endpoint: fmt.Sprintf("%s/bot%s/", apiServer, token),
	}
}

type TG struct {
	http     httpClient
	endpoint string
}

func (t *TG) makeMessage(chatID int64, text string) (io.Reader, error) {
	chat := &Chat{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "markdown",
	}

	body, err := json.Marshal(chat)
	if err != nil {
		return nil, fmt.Errorf("makeMessage: json: %w", err)
	}

	return bytes.NewReader(body), nil
}

func (t *TG) makeRequest(ctx context.Context, method apiMethod, reader io.Reader) (*http.Request, error) {
	url := t.endpoint + string(method)

	req, err := http.NewRequestWithContext(ctx, "POST", url, reader)
	if err != nil {
		return nil, fmt.Errorf("makeMessage: request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func (t *TG) makeResponse(resp *http.Response, result interface{}) error {
	apiResp := &APIResponse{
		Result: result,
		APIResponseError: APIResponseError{
			Ok:          false,
			ErrorCode:   0,
			Description: "",
			Parameters: struct {
				RetryAfter int "json:\"retry_after,omitempty\""
			}{},
		},
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(apiResp); err != nil {
		return fmt.Errorf("makeResponse: body: %w", err)
	}

	if !apiResp.Ok {
		return apiResp.APIResponseError
	}

	return nil
}

func (t *TG) GetMe(ctx context.Context) (*User, error) {
	req, err := t.makeRequest(ctx, apiMethodGetMe, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GetMe: http: %w", err)
	}

	user := &User{
		ID:        0,
		FirstName: "",
		UserName:  "",
	}

	if err := t.makeResponse(resp, user); err != nil {
		return nil, err
	}

	return user, nil
}

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

	message := &Message{
		MessageID: 0,
		Date:      0,
	}

	if err := t.makeResponse(resp, message); err != nil {
		return nil, err
	}

	return message, nil
}
