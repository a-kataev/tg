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

type TGUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	UserName  string `json:"username,omitempty"`
}

type TGChat struct {
	ChatID    int64  `json:"chat_id,omitempty"`
	Text      string `json:"text,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type TGMessage struct {
	MessageID int `json:"message_id"`
	Date      int `json:"date"`
}

type TGAPIResponse struct {
	Result interface{} `json:"result,omitempty"`
	TGAPIResponseError
}

type TGAPIResponseError struct {
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  struct {
		RetryAfter int `json:"retry_after,omitempty"`
	} `json:"parameters,omitempty"`
}

func (r TGAPIResponseError) Error() string {
	return r.Description
}

type tgAPIMethod string

const (
	methodGetMe       tgAPIMethod = "getMe"
	methodSendMessage tgAPIMethod = "sendMessage"
)

type TG interface {
	GetMe(ctx context.Context) (*TGUser, error)
	SendMessage(ctx context.Context, chatID int64, message string) (*TGMessage, error)
}

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
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

type tg struct {
	http     httpClient
	endpoint string
}

func (_ *tg) makeMessage(chatID int64, text string) (io.Reader, error) {
	chat := &TGChat{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "markdown",
	}

	body, err := json.Marshal(chat)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(body), nil
}

func (t *tg) makeRequest(ctx context.Context, method tgAPIMethod, reader io.Reader) (*http.Request, error) {
	url := t.endpoint + string(method)
	req, err := http.NewRequestWithContext(ctx, "POST", url, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func (_ *tg) makeResponse(resp *http.Response, result interface{}) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	apiResp := &TGAPIResponse{
		Result: result,
	}

	if err := json.Unmarshal(body, apiResp); err != nil {
		return err
	}

	if !apiResp.Ok {
		return apiResp.TGAPIResponseError
	}

	return nil
}

func (t *tg) GetMe(ctx context.Context) (*TGUser, error) {
	req, err := t.makeRequest(ctx, methodGetMe, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, err
	}

	tgUser := &TGUser{}

	if err := t.makeResponse(resp, tgUser); err != nil {
		return nil, err
	}

	return tgUser, nil
}

func (t *tg) SendMessage(ctx context.Context, chatID int64, message string) (*TGMessage, error) {
	msg, err := t.makeMessage(chatID, message)
	if err != nil {
		return nil, err
	}

	req, err := t.makeRequest(ctx, methodSendMessage, msg)
	if err != nil {
		return nil, err
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, err
	}

	tgMessage := &TGMessage{}

	if err := t.makeResponse(resp, tgMessage); err != nil {
		return nil, err
	}

	return tgMessage, nil
}
