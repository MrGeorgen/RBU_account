package main
import (
	"time"
	"net/http"
	"bytes"
	"github.com/cornelk/hashmap"
)
type loginStruct struct {
	FalsePassword bool
}
var sessions hashmap.HashMap
const sessionName string = "session"
func login(w http.ResponseWriter, r *http.Request) {
	var err error
	loginStruct := loginStruct{}
	var login bool = false
	if loggedIn(r) {
		http.Redirect(w, r, "dash", http.StatusSeeOther)
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
				Expires: time.Now().Add(10 * 24 * time.Hour),
				HttpOnly: true,
				Secure: true,
			}
			http.SetCookie(w, &cookie)
			sessions.Set(key, username)
			http.Redirect(w, r, "dash", http.StatusSeeOther)
		} else {
			w.Header().Set("Content-Type", "text/html")
			err = loginTmpl.Execute(w, loginStruct)
			log(err)
		}
	} else {
		w.Header().Set("Content-Type", "text/html")
		loginTmpl.Execute(w, loginStruct)
		log(err)
	}
}

func loggedIn(r *http.Request) bool {
	key, err := r.Cookie(sessionName)
	if err != nil {
		return false
	}
	_, valid := sessions.GetStringKey(key.Value)
	return valid
}
