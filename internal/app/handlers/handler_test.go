package handlers

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-url-shortener/internal/app/storage"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader, cookie *http.Cookie) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	require.NoError(t, err)
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
			//return RedirectAttemptedError
		},
	}
	if cookie != nil {
		req.AddCookie(cookie)
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
		db          map[storage.URL]storage.URL
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
			db: map[storage.URL]storage.URL{
				"c101c693": "https://stackoverflow.com/questions/24886015/how-to-convert-uint32-to-string",
			},
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
			repo := storage.NewMemoryMap()
			if tt.db != nil {
				//tt.db = &storage.MemoryMap{Urls: make(map[storage.URL]storage.URL), Mutex: &sync.RWMutex{}}
				for short, long := range tt.db {
					repo.SetLongURL(long, short, "")
				}
			}

			r := NewMainHandler(repo, "http://localhost:8080/")
			ts := httptest.NewServer(r)
			defer ts.Close()

			var body io.Reader = nil
			if tt.requestBody != "" {
				body = strings.NewReader(tt.requestBody)
			}
			if tt.target == "" {
				tt.target = "/"
			}

			resp, respBody := testRequest(t, ts, tt.method, tt.target, body, nil)
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

func TestMainHandlerApi(t *testing.T) {

	type want struct {
		body        string
		contentType string
		statusCode  int
	}
	type record struct {
		ShortURL storage.URL
		LongURL  storage.URL
		UserID   string
	}

	tests := []struct {
		name        string
		method      string
		target      string
		db          []record
		requestBody string
		want        want
	}{
		{
			name:   "Test case #1 JSON",
			method: http.MethodGet,
			target: "/api/user/urls",
			want: want{
				contentType: "application/json; charset=utf-8",
				statusCode:  200,
				body: `[{"short_url":"http://localhost:8080/ac5a78ac","original_url":"https://ya.ru/1123333"},
						{"short_url":"http://localhost:8080/b3f51159","original_url":"https://ya.ru/1123"}]`,
			},
			db: []record{
				{UserID: "370230df-159e-4aec-9f18-922f9c0be328", ShortURL: "ac5a78ac",
					LongURL: "https://ya.ru/1123333"},
				{UserID: "370230df-159e-4aec-9f18-922f9c0be328", ShortURL: "b3f51159",
					LongURL: "https://ya.ru/1123"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := storage.NewMemoryMap()
			if tt.db != nil {
				for _, rec := range tt.db {
					repo.SetLongURL(rec.LongURL, rec.ShortURL, rec.UserID)
				}
			}

			r := NewMainHandler(repo, "http://localhost:8080/")
			ts := httptest.NewServer(r)
			defer ts.Close()

			authCookieVal := `eyJVc2VySUQiOiIzNzAyMzBkZi0xNTllLTRhZWMtOWYxOC05MjJmOWMwYmUzMjgiLCJTaWduIjoiMmJCakJNb2I3cEExWnptMDF4ZjJNK3pWeGhDWFZZK2tQbXpqaWFXSzBrZz0ifQ==`
			authCookie := &http.Cookie{
				Name:  "auth",
				Value: authCookieVal,
			}
			var body io.Reader = nil
			if tt.requestBody != "" {
				body = strings.NewReader(tt.requestBody)
			}
			if tt.target == "" {
				tt.target = "/"
			}

			resp, respBody := testRequest(t, ts, tt.method, tt.target, body, authCookie)
			resp.Body.Close() // statictest: internal/app/handlers/handler_test.go:111:33: response body must be closed
			// проверяем код ответа
			assert.Equal(t, tt.want.statusCode, resp.StatusCode)
			// проверяем код заголовок
			assert.Equal(t, tt.want.contentType, resp.Header.Get("Content-Type"))

			// получаем и проверяем тело запроса
			switch resp.Header.Get("Content-Type") {
			case "application/json; charset=utf-8":
				//assert.JSONEq(t, tt.want.body, respBody,
				//	"Expected body [%s], got [%s]", tt.want.body, respBody)

				//workaround: compare json without order
				var wantJSON GetUserUrlsJSONResponse
				var respJSON GetUserUrlsJSONResponse
				err := json.Unmarshal([]byte(tt.want.body), &wantJSON)
				assert.NoError(t, err)
				err = json.Unmarshal([]byte(respBody), &respJSON)
				assert.NoError(t, err)
				assert.ElementsMatch(t, wantJSON, respJSON)
			default:
				assert.Equal(t, tt.want.body, respBody,
					"Expected body [%s], got [%s]", tt.want.body, respBody)
			}

		})
	}
}
