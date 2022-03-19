package handlers

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-url-shortener/internal/app/storage"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

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
			db: &storage.DB{Urls: map[storage.URL]storage.URL{
				"c101c693": "https://stackoverflow.com/questions/24886015/how-to-convert-uint32-to-string",
			}, Mutex: &sync.Mutex{}},
			want: want{
				contentType: "text/html; charset=utf-8",
				statusCode:  307,
				body:        "<a href=\"https://stackoverflow.com/questions/24886015/how-to-convert-uint32-to-string\">Temporary Redirect</a>.\n\n",
			},
		},
		{
			name:   "Test case #2",
			method: http.MethodGet,
			target: "/c101c693",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  404,
				body:        "404 page not found\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var body io.Reader = nil
			if tt.requestBody != "" {
				body = strings.NewReader(tt.requestBody)
			}
			if tt.target == "" {
				tt.target = "/"
			}
			request := httptest.NewRequest(tt.method, tt.target, body)

			// создаём новый Recorder
			w := httptest.NewRecorder()
			// определяем хендлер
			if tt.db == nil {
				var mutex sync.Mutex
				tt.db = &storage.DB{Urls: make(map[storage.URL]storage.URL), Mutex: &mutex}
			}

			h := MainHandler{
				Repository: tt.db,
				Location:   "http://localhost:8080/",
			}
			// запускаем сервер
			h.ServeHTTP(w, request)
			res := w.Result()

			// проверяем код ответа
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			// проверяем код заголовок
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))

			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want.body, string(resBody),
				"Expected body [%s], got [%s]", tt.want.body, w.Body.String())
		})
	}
}
