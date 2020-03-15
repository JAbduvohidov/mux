package authenticated

import (
	"context"
	"net/http"
)

func Authenticated(authenticated func(ctx context.Context) bool) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			if !authenticated(request.Context()) {
				http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			next(writer, request)
		}
	}
}

