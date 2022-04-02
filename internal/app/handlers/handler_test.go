package handlers

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-url-shortener/internal/app/storage"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	require.NoError(t, err)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
			//return RedirectAttemptedError
		},
	}
	//resp, err := http.DefaultClient.Do(req)
	resp, err := client.Do(req)

	require.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestMainHandler_ServeHTTP(t *testing.T) {

	type want struct {
		body        string
		contentType string
		statusCode  int
	}
	tests := []struct {
		name        string
		method      string
		target      string
		db          storage.Repository
		requestBody string
		want        want
	}{
		// TODO: Add test cases.
		{
			name:        "Test case #1",
			method:      http.MethodPost,
			requestBody: "https://stackoverflow.com/questions/24886015/how-to-convert-uint32-to-string",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  201,
				body:        "http://localhost:8080/c101c693",
			},
		},
		{
			name:   "Test case #2",
			method: http.MethodGet,
			target: "/c101c693",
			db: &storage.MemoryMap{Urls: map[storage.URL]storage.URL{
				"c101c693": "https://stackoverflow.com/questions/24886015/how-to-convert-uint32-to-string",
			}, Mutex: &sync.Mutex{}},
			want: want{
				contentType: "text/html; charset=utf-8",
				statusCode:  307,
				body:        "<a href=\"https://stackoverflow.com/questions/24886015/how-to-convert-uint32-to-string\">Temporary Redirect</a>.\n\n",
			},
		},
		{
			name:   "Test case #3",
			method: http.MethodGet,
			target: "/c101c693",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  404,
				body:        "404 page not found\n",
			},
		},
		{
			name:        "Test case #4 with json",
			method:      http.MethodPost,
			target:      "/api/shorten",
			requestBody: `{"url": "https://practicum.yandex.ru/learn/go-developer"}`,
			want: want{
				contentType: "application/json; charset=utf-8",
				statusCode:  201,
				body:        `{"result": "http://localhost:8080/8d34fd6f"}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.db == nil {
				var mutex sync.Mutex
				tt.db = &storage.MemoryMap{Urls: make(map[storage.URL]storage.URL), Mutex: &mutex}
			}

			r := NewMainHandler(tt.db, "http://localhost:8080/")
			ts := httptest.NewServer(r)
			defer ts.Close()

			var body io.Reader = nil
			if tt.requestBody != "" {
				body = strings.NewReader(tt.requestBody)
			}
			if tt.target == "" {
				tt.target = "/"
			}

			resp, respBody := testRequest(t, ts, tt.method, tt.target, body)
			resp.Body.Close() // statictest: internal/app/handlers/handler_test.go:111:33: response body must be closed
			// проверяем код ответа
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
			// проверяем код заголовок
			assert.Equal(t, tt.want.contentType, resp.Header.Get("Content-Type"))

			// получаем и проверяем тело запроса
			switch resp.Header.Get("Content-Type") {
			case "application/json; charset=utf-8":
				assert.JSONEq(t, tt.want.body, respBody,
					"Expected body [%s], got [%s]", tt.want.body, respBody)
			default:
				assert.Equal(t, tt.want.body, respBody,
					"Expected body [%s], got [%s]", tt.want.body, respBody)
			}

		})
	}
}
