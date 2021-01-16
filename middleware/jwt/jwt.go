package jwt

import (
	"context"
	"github.com/JAbduvohidov/jwt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type ContextKey string

var payloadContextKey = ContextKey("jwt")

const (
	SourceAuthorization = iota
	SourceCookie
)

func JWT(source int, payloadType reflect.Type, secret jwt.Secret) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			token := ""

			switch source {
			case SourceAuthorization:
				// TODO: move to func
				header := request.Header.Get("Authorization")
				if header == "" {
					break
				}
				if !strings.HasPrefix(header, "Bearer ") {
					break
				}
				token = header[len("Bearer "):]
			case SourceCookie:
				// TODO: move to func
				cookie, err := request.Cookie("token")
				if err != nil {
					if err == http.ErrNoCookie {
						break
					}
					break
				}
				token = cookie.Value
			}

			if token == "" {
				next(writer, request)
				return
			}

			ok, err := jwt.Verify(token, secret)
			if err != nil {
				http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			if !ok {
				http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			payload := reflect.New(payloadType).Interface()

			err = jwt.Decode(token, payload)
			if err != nil {
				http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			ok, err = jwt.IsNotExpired(payload, time.Now())
			if err != nil {
				http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			if !ok {
				http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			log.Print(payload)

			ctx := context.WithValue(request.Context(), payloadContextKey, payload)
			next(writer, request.WithContext(ctx))
		}
	}
}

func FromContext(ctx context.Context) (payload interface{}) {
	payload = ctx.Value(payloadContextKey)
	return
}

func IsContextNonEmpty(ctx context.Context) bool {
	return nil != ctx.Value(payloadContextKey)
}
