package handlers

import (
	"net/http"

	mw "github.com/sakid00/massmaker-be/internal/http/middleware"
)

func userFrom(r *http.Request) (string, string) {
	return mw.UserFromCtx(r.Context())
}
