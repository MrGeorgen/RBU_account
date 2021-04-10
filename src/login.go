package main
import (
	"time"
	"net/http"
	"bytes"
	"sync"
)
type loginStruct struct {
	FalsePassword bool
}
var sessions sync.Map
const sessionName string = "session"
const sessionTimeout time.Duration = 10 * 24 * time.Hour
func login(w http.ResponseWriter, r *http.Request) {
	var redirectUrl = r.FormValue("redirecturl")
	if redirectUrl == "" {
		redirectUrl = "/"
	}
	loginStruct := loginStruct{}
	var login bool = false
	if loggedIn(r) {
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodPost {
		var username string = r.FormValue("username")
		var password string = r.FormValue("password")
		var hash []byte
		var salt []byte
		db.QueryRow("SELECT hash,salt FROM account WHERE username = ?", username).Scan(&hash, &salt)
		login = bytes.Equal(hashFunc([]byte(password), salt), hash)
		loginStruct.FalsePassword = !login
		if login {
			key, err := GenerateRandomString(64)
			log(err)
			cookie := http.Cookie{
				Name: sessionName,
				Value: key,
				Expires: time.Now().Add(sessionTimeout),
				HttpOnly: true,
				Secure: true,
			}
			http.SetCookie(w, &cookie)
			sessions.Store(key, username)
			go deleteSession(key)
			http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		}
	}
	if r.Method == http.MethodGet || !login {
		runTemplate(r, w, loginTmpl, loginStruct)
	}
}

func loggedIn(r *http.Request) bool {
	cookie, err := r.Cookie(sessionName)
	if err != nil {
		return false
	}
	_, valid := sessions.Load(cookie.Value)
	return valid
}

func deleteSession(key string) {
	time.Sleep(sessionTimeout)
	sessions.Delete(key)
}
