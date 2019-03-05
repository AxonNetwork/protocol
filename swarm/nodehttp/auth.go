package nodehttp

import (
	"net/http"
)

func BasicAuth(username, password string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

		reqUsername, reqPassword, authOK := r.BasicAuth()
		if authOK == false {
			http.Error(w, "Not authorized", 401)
			return
		}

		if reqUsername != username || reqPassword != password {
			http.Error(w, "Not authorized", 401)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
