//go:generate mockery --name httpClient --structname mockHTTPClient --inpackage --filename tg_mock_test.go

package tg

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
)

func newTestTG() *TG {
	return &TG{
		http:      nil,
		endpoint:  "",
		parseMode: "",
	}
}

func Test_makeRequest(t *testing.T) {
	testTG := newTestTG()

	t.Parallel()

	t.Run("1", func(t *testing.T) {
		t.Parallel()
		request, err := testTG.makeRequest(nil, apiMethod(""), nil) //nolint
		assert.Nil(t, request)
		assert.EqualError(t, err, "makeMessage: request: net/http: nil Context")
	})

	t.Run("2", func(t *testing.T) {
		t.Parallel()
		request, err := testTG.makeRequest(context.Background(), apiMethod(""), nil)
		assert.IsType(t, &http.Request{}, request) //nolint
		assert.Nil(t, err)
	})
}

var errTest = errors.New("test")

type errReader struct{}

func (e *errReader) Read(p []byte) (int, error) {
	return 0, errTest
}

func (e *errReader) Close() error {
	return errTest
}

func Test_makeResponse_BadReader(t *testing.T) {
	testTG := newTestTG()

	t.Parallel()
	err := testTG.makeResponse(&http.Response{ //nolint
		Body: &errReader{},
	}, nil)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "makeResponse: body: test")
}

func Test_makeResponse_Cases(t *testing.T) {
	testTG := newTestTG()

	t.Parallel()

	type table struct {
		responseBody []byte
		clientError  string
	}

	tables := []table{
		{
			responseBody: []byte{},
			clientError:  "makeResponse: body: EOF",
		},
		{
			responseBody: []byte("test"),
			clientError:  "makeResponse: body: invalid character 'e' in literal true (expecting 'r')",
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
			clientError:  "makeResponse: body: json: cannot unmarshal string into Go struct field APIResponse.ok of type bool",
		},
	}

	test := func(tn int, table table) {
		t.Run(strconv.Itoa(tn), func(t *testing.T) {
			t.Parallel()
			clientResponse := &http.Response{ //nolint
				Body: io.NopCloser(bytes.NewBuffer(table.responseBody)),
			}

			err := testTG.makeResponse(clientResponse, nil)
			assert.EqualErrorf(t, err, table.clientError, "%d", tn)
		})
	}

	for tn, table := range tables {
		test(tn, table)
	}
}

func Test_makeResponse_Update(t *testing.T) {
	testTG := newTestTG()

	testUser := &User{
		ID:        1,
		FirstName: "test",
		UserName:  "",
	}
	updateUser := &User{
		ID:        0,
		FirstName: "",
		UserName:  "",
	}

	t.Parallel()

	err := testTG.makeResponse(
		&http.Response{ //nolint
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{"ok":true,"result":{"id":1,"first_name":"test"}}`))),
		}, updateUser)
	assert.Nil(t, err)
	assert.Equal(t, updateUser, testUser)
}

func Test_makeResponse_OK(t *testing.T) {
	testTG := newTestTG()

	t.Parallel()

	err := testTG.makeResponse(
		&http.Response{ //nolint
			Body: io.NopCloser(bytes.NewBuffer([]byte(`{"ok":true}`))),
		}, nil)
	assert.Nil(t, err)
}

func Test_GetMe(t *testing.T) {
	t.Parallel()

	t.Run("1", func(t *testing.T) {
		t.Parallel()
		testTG := newTestTG()
		user, err := testTG.GetMe(nil) //nolint
		assert.EqualError(t, err, "makeMessage: request: net/http: nil Context")
		assert.Nil(t, user)
	})

	t.Run("2", func(t *testing.T) {
		t.Parallel()
		testTG := newTestTG()
		testHTTPClient := &mockHTTPClient{} //nolint
		testHTTPClient.On("Do", mock.Anything, mock.Anything).Return(nil, errTest)
		testTG.http = testHTTPClient
		user, err := testTG.GetMe(context.Background())
		assert.EqualError(t, err, "GetMe: http: test")
		assert.Nil(t, user)
	})

	t.Run("3", func(t *testing.T) {
		t.Parallel()
		testTG := newTestTG()
		testHTTPClient := &mockHTTPClient{} //nolint
		testHTTPClient.On("Do", mock.Anything, mock.Anything).Return(
			&http.Response{ //nolint
				Body: &errReader{},
			}, nil)
		testTG.http = testHTTPClient
		user, err := testTG.GetMe(context.Background())
		assert.EqualError(t, err, "makeResponse: body: test")
		assert.Nil(t, user)
	})

	t.Run("4", func(t *testing.T) {
		t.Parallel()
		testTG := newTestTG()
		testUser := &User{
			ID:        1,
			FirstName: "test",
			UserName:  "",
		}
		testHTTPClient := &mockHTTPClient{} //nolint
		testHTTPClient.On("Do", mock.Anything, mock.Anything).Return(
			&http.Response{ //nolint
				Body: io.NopCloser(bytes.NewBuffer([]byte(`{"ok":true,"result":{"id":1,"first_name":"test"}}`))),
			}, nil)
		testTG.http = testHTTPClient
		user, err := testTG.GetMe(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, user, testUser)
	})
}
