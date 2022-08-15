package http

import "net/http"

func (s Server) IsAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := r.Header.Get("X-API-Token"); token != "" {
			// check header
			if !s.apiService.ValidateAPIKey(r.Context(), token) {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

		} else if key := r.URL.Query().Get("apikey"); key != "" {
			// check query param lke ?apikey=TOKEN
			if !s.apiService.ValidateAPIKey(r.Context(), key) {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
		} else {
			// check session
			session, _ := s.cookieStore.Get(r, "user_session")

			// Check if user is authenticated
			if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
