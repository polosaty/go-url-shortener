package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"net/http"
)

type Session struct {
	UserID string
	Sign   []byte
}

func makeUserId() string {
	return uuid.New().String()
}

func NewSession() *Session {
	return &Session{
		UserID: makeUserId(),
	}
}
func (s *Session) makeSignature(secretKey []byte) []byte {
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(s.UserID))
	return h.Sum(nil)
}

func (s *Session) signSession(secretKey []byte) {
	s.Sign = s.makeSignature(secretKey)
}

func (s *Session) checkSignature(secretKey []byte) bool {
	return hmac.Equal(s.Sign, s.makeSignature(secretKey))
}

func authMiddleware(secretKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var session *Session
			cookie, err := r.Cookie("auth")
			if err != nil {
				if errors.Is(err, http.ErrNoCookie) {
					session = NewSession()
					session.signSession(secretKey)
					sessionJson, err := json.Marshal(session)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						io.WriteString(w, err.Error())
						return
					}

					cookie = &http.Cookie{
						Name:  "auth",
						Value: base64.URLEncoding.EncodeToString(sessionJson),
						//Expires: time.Now().Add(48 * time.Hour),
					}
					r.AddCookie(cookie)
					http.SetCookie(w, cookie)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					io.WriteString(w, err.Error())
					return
				}
			} else {

				cookieJson, err := base64.URLEncoding.DecodeString(cookie.Value)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					io.WriteString(w, err.Error())
					return
				}
				err = json.Unmarshal(cookieJson, &session)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					io.WriteString(w, err.Error())
					return
				}
				if !session.checkSignature(secretKey) {
					w.WriteHeader(http.StatusBadRequest)
					io.WriteString(w, "bad session signature")
					return
				}
			}

			r = r.WithContext(context.WithValue(r.Context(), interface{}("Session"), session))

			next.ServeHTTP(w, r)
		})
	}
}

func GetSession(req *http.Request) *Session {
	sessCtx := req.Context().Value("Session")
	sess, _ := sessCtx.(*Session)
	return sess
}
