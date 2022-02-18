//go:generate mockery --name httpClient --structname mockHTTPClient --inpackage --filename tg_mock_test.go

package tg

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_makeRequest(t *testing.T) {
	testTG := &tg{}

	request, err := testTG.makeRequest(nil, tgAPIMethod(""), nil) //nolint
	assert.Nil(t, request)
	assert.EqualError(t, err, "net/http: nil Context")

	request, err = testTG.makeRequest(context.Background(), tgAPIMethod(""), nil)
	assert.IsType(t, &http.Request{}, request)
	assert.Nil(t, err)
}

type errReader struct{}

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test")
}

func (e *errReader) Close() error {
	return errors.New("test")
}

func Test_makeResponse_BadReader(t *testing.T) {
	testTG := &tg{}

	err := testTG.makeResponse(&http.Response{
		Body: &errReader{},
	})
	assert.NotNil(t, err)
	assert.EqualError(t, err, "test")
}

func Test_makeResponse_Cases(t *testing.T) {
	testTG := &tg{}

	tables := []struct {
		responseBody []byte
		clientError  string
	}{
		{
			responseBody: []byte{},
			clientError:  "unexpected end of JSON input",
		},
		{
			responseBody: []byte("test"),
			clientError:  "invalid character 'e' in literal true (expecting 'r')",
		},
		{
			responseBody: []byte("{}"),
			clientError:  "",
		},
		{
			responseBody: []byte(`{"description":"test"}`),
			clientError:  "test",
		},
		{
			responseBody: []byte(`{"ok":"test"}`),
			clientError:  "json: cannot unmarshal string into Go struct field TGAPIResponse.ok of type bool",
		},
	}

	for tt, table := range tables {
		clientResponse := &http.Response{
			Body: ioutil.NopCloser(bytes.NewBuffer(table.responseBody)),
		}

		err := testTG.makeResponse(clientResponse)
		assert.EqualErrorf(t, err, table.clientError, "%d", tt)
	}
}

func Test_makeResponse_OK(t *testing.T) {
	testTG := &tg{}

	err := testTG.makeResponse(&http.Response{
		Body: ioutil.NopCloser(bytes.NewBuffer([]byte(`{"ok":true}`))),
	})
	assert.Nil(t, err)
}

func Test_GetMe(t *testing.T) {
	testTG := &tg{}

	err := testTG.GetMe(nil) //nolint
	assert.EqualError(t, err, "net/http: nil Context")

	testHTTPClient := &mockHTTPClient{}
	testHTTPClient.On("Do", mock.Anything, mock.Anything).Return(nil, errors.New("test"))
	testTG.http = testHTTPClient
	err = testTG.GetMe(context.Background())
	assert.EqualError(t, err, "test")

	testHTTPClient = &mockHTTPClient{}
	testHTTPClient.On("Do", mock.Anything, mock.Anything).Return(&http.Response{
		Body: &errReader{},
	}, nil)
	testTG.http = testHTTPClient
	err = testTG.GetMe(context.Background())
	assert.EqualError(t, err, "test")
}
