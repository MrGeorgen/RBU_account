package main
import (
	"net/http"
	"encoding/json"
)
type accountApiResponse struct {
	Username string `json:"username"`
	DiscordUserId string `json:"discordUserId"`
	Email string `json:"email"`
}
func accountApi(w http.ResponseWriter, r *http.Request) {
	var accountKey string = r.FormValue("accountkey")
	var password string = r.FormValue("password")
	if password != secret.ApiToken {
		http.Error(w, "Error 401 false password", 401)
		return
	}
	var account accountApiResponse
	var success bool
	var usernameInter interface{}
	usernameInter, success = sessions.GetStringKey(accountKey)
	account.Username = usernameInter.(string)
	if !success {
		http.Error(w, "Error 400 invalid session", 400)
	}
	db.QueryRow("SELECT email,discordUserId FROM account WHERE username = ?", account.Username).Scan(&account.Email, &account.DiscordUserId)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}
