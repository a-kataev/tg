//nolint:exhaustruct
package tg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

//nolint:gochecknoglobals,gosec
var (
	testBadParseMode = ParseMode("test")

	testText    = string(make([]byte, rand.Intn(MaxTextSize-1)+1))
	testBadText = string(make([]byte, MaxTextSize+1))

	testToken = "1:test"
)

func Test_ParseMode_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc   string
		mode   ParseMode
		result error
	}{
		{
			desc:   ErrUnknownParseMode.Error(),
			mode:   testBadParseMode,
			result: ErrUnknownParseMode,
		},
		{
			desc:   "nil_result",
			mode:   MarkdownParseMode,
			result: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.mode.Validate(), test.result)
		})
	}
}

func Test_BaseMessage_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc   string
		msg    func() *BaseMessage
		result error
	}{
		{
			desc:   ErrEmptyChatID.Error(),
			msg:    func() *BaseMessage { return &BaseMessage{} },
			result: ErrEmptyChatID,
		},
		{
			desc: ErrEmptyText.Error(),
			msg: func() *BaseMessage {
				return &BaseMessage{
					ChatID: 1,
				}
			},
			result: ErrEmptyText,
		},
		{
			desc: ErrTextTooLong.Error(),
			msg: func() *BaseMessage {
				return &BaseMessage{
					ChatID: 1,
					Text:   testBadText,
				}
			},
			result: ErrTextTooLong,
		},
		{
			desc: ErrUnknownParseMode.Error(),
			msg: func() *BaseMessage {
				return &BaseMessage{
					ChatID:    1,
					Text:      testText,
					ParseMode: testBadParseMode,
				}
			},
			result: ErrUnknownParseMode,
		},
		{
			desc: "nil_result",
			msg: func() *BaseMessage {
				return &BaseMessage{
					ChatID:    1,
					Text:      testText,
					ParseMode: HTMLParseMode,
				}
			},
			result: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.msg().Validate(), test.result)
		})
	}
}

func Test_SendMessage_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc   string
		msg    func() *SendMessage
		result error
	}{
		{
			desc: ErrIncorrectMessageThreadID.Error(),
			msg: func() *SendMessage {
				return &SendMessage{
					BaseMessage: BaseMessage{
						ChatID:    1,
						Text:      testText,
						ParseMode: HTMLParseMode,
					},
					MessageThreadID: -1,
				}
			},
			result: ErrIncorrectMessageThreadID,
		},
		{
			desc:   ErrEmptyChatID.Error(),
			msg:    func() *SendMessage { return &SendMessage{} },
			result: ErrEmptyChatID,
		},
		{
			desc: "nil_result",
			msg: func() *SendMessage {
				return &SendMessage{
					BaseMessage: BaseMessage{
						ChatID:    1,
						Text:      testText,
						ParseMode: MarkdownV2ParseMode,
					},
					MessageThreadID: 0,
				}
			},
			result: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.msg().Validate(), test.result)
		})
	}
}

func Test_EditMessage_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc   string
		msg    func() *EditMessage
		result error
	}{
		{
			desc: ErrIncorrectMessageID.Error(),
			msg: func() *EditMessage {
				return &EditMessage{
					MessageID: 0,
					BaseMessage: BaseMessage{
						ChatID:    1,
						Text:      testText,
						ParseMode: HTMLParseMode,
					},
				}
			},
			result: ErrIncorrectMessageID,
		},
		{
			desc:   ErrIncorrectMessageID.Error(),
			msg:    func() *EditMessage { return &EditMessage{} },
			result: ErrIncorrectMessageID,
		},
		{
			desc: "nil_result",
			msg: func() *EditMessage {
				return &EditMessage{
					MessageID: 1,
					BaseMessage: BaseMessage{
						ChatID:    1,
						Text:      testText,
						ParseMode: HTMLParseMode,
					},
				}
			},
			result: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.msg().Validate(), test.result)
		})
	}
}

func Test_DeleteMessage_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc   string
		msg    func() *DeleteMessage
		result error
	}{
		{
			desc:   ErrEmptyChatID.Error(),
			msg:    func() *DeleteMessage { return &DeleteMessage{} },
			result: ErrEmptyChatID,
		},
		{
			desc: ErrIncorrectMessageID.Error(),
			msg: func() *DeleteMessage {
				return &DeleteMessage{
					ChatID: 1,
				}
			},
			result: ErrIncorrectMessageID,
		},
		{
			desc: "nil_result",
			msg: func() *DeleteMessage {
				return &DeleteMessage{
					ChatID:    1,
					MessageID: 1,
				}
			},
			result: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.msg().Validate(), test.result)
		})
	}
}

func Test_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc   string
		value  any
		result error
	}{
		{
			desc:   ErrValueNil.Error(),
			value:  nil,
			result: ErrValueNil,
		},
		{
			desc:   ErrValueNotPtr.Error(),
			value:  "",
			result: ErrValueNotPtr,
		},
		{
			desc:   ErrValueNotStructOrBool.Error(),
			value:  reflect.New(reflect.TypeOf("")).Interface(),
			result: ErrValueNotStructOrBool,
		},
		{
			desc:   "err_result",
			value:  reflect.New(reflect.TypeOf(struct{}{})).Interface(),
			result: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, validate(test.value), test.result)
		})
	}
}

func Test_NewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		token   string
		options func() []Option
		result  error
	}{
		{
			desc:    ErrIncorrentToken.Error(),
			token:   "test",
			options: func() []Option { return []Option{} },
			result:  ErrIncorrentToken,
		},
		{
			desc:  "empty_url",
			token: testToken,
			options: func() []Option {
				return []Option{
					WithAPIServer(""),
				}
			},
			result: fmt.Errorf("apiserver: %w",
				&url.Error{
					Op:  "parse",
					URL: "",
					Err: errors.New("empty url"), //nolint:goerr113
				},
			),
		},
		{
			desc:  ErrIncorrectScheme.Error(),
			token: testToken,
			options: func() []Option {
				return []Option{
					WithAPIServer("test://"),
				}
			},
			result: fmt.Errorf("apiserver: url: %w", ErrIncorrectScheme),
		},
		{
			desc:  ErrEmptyHost.Error(),
			token: testToken,
			options: func() []Option {
				return []Option{
					WithAPIServer("http://"),
				}
			},
			result: fmt.Errorf("apiserver: url: %w", ErrEmptyHost),
		},
		{
			desc:  ErrHTTPClientNil.Error(),
			token: testToken,
			options: func() []Option {
				return []Option{
					WithAPIServer("http://test"),
					WithHTTPClient(nil),
				}
			},
			result: ErrHTTPClientNil,
		},
		{
			desc:  "err_return_options",
			token: testToken,
			options: func() []Option {
				return []Option{
					WithAPIServer("http://test"),
					WithHTTPClient(defaultHTTPClient),
				}
			},
			result: nil,
		},
		{
			desc:  "err_return",
			token: testToken,
			options: func() []Option {
				return []Option{}
			},
			result: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			opts := test.options()
			_, err := NewClient(test.token, opts...)

			assert.Equal(t, errors.Unwrap(err), test.result)
		})
	}
}

var errTest = errors.New("test")

type errReader struct{}

func (e *errReader) Read(_ []byte) (int, error) {
	return 0, errTest
}

func (e *errReader) Close() error {
	return errTest
}

//nolint:funlen
func Test_Client_API(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc      string
		http      func() HTTPClient
		req, resp func() any
		result    error
	}{
		{
			desc:   ErrValueNotPtr.Error(),
			http:   func() HTTPClient { return nil },
			req:    func() any { return 1 },
			resp:   func() any { return nil },
			result: fmt.Errorf("validate: req %w", ErrValueNotPtr),
		},
		{
			desc:   ErrValueNil.Error(),
			http:   func() HTTPClient { return nil },
			req:    func() any { return nil },
			resp:   func() any { return nil },
			result: fmt.Errorf("validate: resp %w", ErrValueNil),
		},
		{
			desc: errTest.Error(),
			http: func() HTTPClient {
				client := &mockHTTPClient{}
				client.On("Do", mock.Anything, mock.Anything).Return(nil, errTest)

				return client
			},
			req: func() any { return nil },
			resp: func() any {
				return reflect.New(reflect.TypeOf(struct{}{})).Interface()
			},
			result: fmt.Errorf("request: %w", errTest),
		},
		{
			desc: errTest.Error(),
			http: func() HTTPClient {
				client := &mockHTTPClient{}
				client.On("Do", mock.Anything, mock.Anything).
					Return(&http.Response{
						Body: &errReader{},
					},
						nil,
					)

				return client
			},
			req: func() any { return nil },
			resp: func() any {
				return reflect.New(reflect.TypeOf(struct{}{})).Interface()
			},
			result: fmt.Errorf("response: json: %w", errTest),
		},
		{
			desc: io.EOF.Error(),
			http: func() HTTPClient {
				client := &mockHTTPClient{}
				client.On("Do", mock.Anything, mock.Anything).
					Return(
						&http.Response{
							Body: io.NopCloser(bytes.NewBuffer([]byte{})),
						},
						nil,
					)

				return client
			},
			req: func() any { return nil },
			resp: func() any {
				return reflect.New(reflect.TypeOf(struct{}{})).Interface()
			},
			result: fmt.Errorf("response: json: %w", io.EOF),
		},
		{
			desc: io.EOF.Error(),
			http: func() HTTPClient {
				client := &mockHTTPClient{}
				client.On("Do", mock.Anything, mock.Anything).
					Return(
						&http.Response{
							Body: io.NopCloser(bytes.NewBuffer([]byte("{}"))), //nolint:mirror
						},
						nil,
					)

				return client
			},
			req: func() any { return nil },
			resp: func() any {
				return reflect.New(reflect.TypeOf(struct{}{})).Interface()
			},
			result: fmt.Errorf("response: %w", ResponseError{}),
		},
		{
			desc: "nil_err",
			http: func() HTTPClient {
				resp := new(Response)
				resp.Ok = true

				body, _ := json.Marshal(resp) //nolint:errchkjson

				client := &mockHTTPClient{}
				client.On("Do", mock.Anything, mock.Anything).Return(
					&http.Response{
						Body: io.NopCloser(bytes.NewBuffer(body)),
					},
					nil,
				)

				return client
			},
			req: func() any { return nil },
			resp: func() any {
				return reflect.New(reflect.TypeOf(struct{}{})).Interface()
			},
			result: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			client := new(Client)
			client.http = test.http()

			assert.Equal(t, client.API(context.Background(), "", test.req(), test.resp()), test.result)
		})
	}
}
