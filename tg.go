package tg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

var tgAPIServer = "https://api.telegram.org"

type TG interface {
	GetMe(ctx context.Context) error
	SendMessage(ctx context.Context, chatID int64, message string) error
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type tg struct {
	http     httpClient
	endpoint string
}

func NewTG(token string) TG {
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 10 * time.Second,
		},
	}

	return NewTGWithClient(token, client)
}

func NewTGWithClient(token string, client *http.Client) TG {
	return &tg{
		http:     client,
		endpoint: fmt.Sprintf("%s/bot%s/", tgAPIServer, token),
	}
}

type tgAPIMethod string

const (
	methodGetMe       tgAPIMethod = "getMe"
	methodSendMessage tgAPIMethod = "sendMessage"
)

func (t *tg) makeRequest(ctx context.Context, method tgAPIMethod, reader io.Reader) (*http.Request, error) {
	url := t.endpoint + string(method)
	req, err := http.NewRequestWithContext(ctx, "POST", url, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

type TGAPIResponse struct {
	Method      string `json:"_method,omitempty"`
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  struct {
		MigrateToChatID int64 `json:"migrate_to_chat_id,omitempty"`
		RetryAfter      int   `json:"retry_after,omitempty"`
	} `json:"parameters,omitempty"`
}

func (r *TGAPIResponse) Error() string {
	return r.Description
}

func (_ *tg) makeResponse(resp *http.Response) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	apiResp := &TGAPIResponse{}

	if err := json.Unmarshal(body, apiResp); err != nil {
		return err
	}

	if !apiResp.Ok {
		return apiResp
	}

	return nil
}

func (t *tg) GetMe(ctx context.Context) error {
	req, err := t.makeRequest(ctx, methodGetMe, nil)
	if err != nil {
		return err
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return err
	}

	return t.makeResponse(resp)
}

type tgAPIMessage struct {
	ChatID    int64  `json:"chat_id,omitempty"`
	Text      string `json:"text,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"`
}

func (_ *tg) makeMessage(chatID int64, text string) (io.Reader, error) {
	msg := &tgAPIMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "markdown",
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(body), nil
}

func (t *tg) SendMessage(ctx context.Context, chatID int64, message string) error {
	msg, err := t.makeMessage(chatID, message)
	if err != nil {
		return err
	}

	req, err := t.makeRequest(ctx, methodSendMessage, msg)
	if err != nil {
		return err
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return err
	}

	return t.makeResponse(resp)
}
