package user

import (
	"fmt"
	"io"
	"net/http"
)

type user struct {
}

func NewFromCookie(cookie string) *user {
	fmt.Println("vim-go")
	u := &user{}
	return u
}
func ShowUserFromCookieHandler(cookieName, authBaseurl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			http.Error(w, "1:"+http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		resp, err := http.Get(authBaseurl + "/token/" + cookie.Value)
		if err != nil || resp.StatusCode != http.StatusOK {
			http.Error(w, "2:"+http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		//w.Write(resp.Body)
		io.Copy(w, resp.Body)
	}
}
