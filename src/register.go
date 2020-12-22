package main
import (
	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"strings"
	"regexp"
	"github.com/dlclark/regexp2"
	"context"
	"time"
	"code.gitea.io/sdk/gitea"
	"crypto/rand"
	"github.com/cornelk/hashmap"
)
type account struct {
	email    string
	username string
	password string
	discordUsername string
	discordTag string
	discordId string
}
type WrongAccount struct {
	User  bool
	Pass  bool
	Email bool
	DiscordUser bool
}
type registertmpl struct {
	Success bool
	WrongAccount WrongAccount
	AlreadyEsitsInDatabase struct{
		Username        bool
		DiscordUsername bool
	}
}
type SubmitStruct struct {
	Success bool
}
var cacheAccounts hashmap.HashMap
var rusername *regexp.Regexp
var remail *regexp2.Regexp
var rpassword *regexp2.Regexp
func register(w http.ResponseWriter, r *http.Request) {
	registerstruct := registertmpl{}
	if r.Method == http.MethodPost {
		var newAccount account
		var newRbuMember *discordgo.Member
		var split = strings.Split(r.FormValue("discordUser"), "#")
		newAccount = account{
			email:    r.FormValue("email"),
			username: r.FormValue("username"),
			password: r.FormValue("password"),
		}
		if len(split) == 2 {
			newAccount.discordUsername = split[0]
			newAccount.discordTag = split[1]
		}
		registerstruct.WrongAccount.Email, _ = remail.MatchString(newAccount.email)
		registerstruct.WrongAccount.Email = !registerstruct.WrongAccount.Email
		registerstruct.WrongAccount.User = !rusername.MatchString(newAccount.username)
		registerstruct.WrongAccount.Pass, _ = rpassword.MatchString(newAccount.password)
		registerstruct.WrongAccount.Pass = !registerstruct.WrongAccount.Pass
		newRbuMember, registerstruct.WrongAccount.DiscordUser = getRbuMember(newAccount.discordUsername, newAccount.discordTag)
		registerstruct.WrongAccount.DiscordUser = !registerstruct.WrongAccount.DiscordUser
		if registerstruct.WrongAccount.DiscordUser {
			goto registerReturn
		}
		newAccount.discordId = newRbuMember.User.ID
		{
			var username string
			registerstruct.AlreadyEsitsInDatabase.Username = db.QueryRow("SELECT username FROM account WHERE username = ?", newAccount.username).Scan(&username) == nil || UsernameExistsInMem(newAccount.username) // check if username exits
			registerstruct.AlreadyEsitsInDatabase.DiscordUsername = db.QueryRow("SELECT username FROM account WHERE discordUserId = ?", newAccount.discordId).Scan(&username) == nil || discordUsernameExistsInMem(newAccount.discordId)
		}
		registerstruct.Success = !registerstruct.WrongAccount.User && !registerstruct.WrongAccount.Pass && !registerstruct.WrongAccount.Email && !registerstruct.WrongAccount.DiscordUser && !registerstruct.AlreadyEsitsInDatabase.DiscordUsername && !registerstruct.AlreadyEsitsInDatabase.Username
		if !registerstruct.Success {
			goto registerReturn
		}
		token, err := GenerateRandomStringURLSafe(64)
		log(err)
		var dmChannel *discordgo.Channel
		dmChannel, err = discord.UserChannelCreate(newRbuMember.User.ID)
		log(err)
		discord.ChannelMessageSend(dmChannel.ID, "Bitte klicke auf den Link, um die Erstellung des Accounts abzuschließen.\nhttp://localhost:8080/submit?token=" + token)
		cacheAccounts.Set(token, newAccount)
	}
	registerReturn: runTemplate(w, registerTmpl, registerstruct)
}
	func submit(w http.ResponseWriter, r *http.Request) {
		var err error
		var submitStruct SubmitStruct
		token := r.FormValue("token")
		var accInter interface{}
		accInter, submitStruct.Success = cacheAccounts.GetStringKey(token)
		if !submitStruct.Success {
			goto submitReturn
		}
		{
			var account account = accInter.(account)
			cacheAccounts.Del(token)
			salt := make([]byte, 32)
			_, err = rand.Read(salt)
			log(err)
			hash := hashFunc([]byte(account.password), salt)
			// add user to the database
			query := "INSERT INTO account(username, email, hash, salt, discordUserId) VALUES (?, ?, ?, ?, ?)"
			ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelfunc()
			stmt, err := db.PrepareContext(ctx, query)
			log(err)
			defer stmt.Close()
			_, err = stmt.ExecContext(ctx, account.username, account.email, hash, salt, account.discordId)
			log(err)
			//_, err = moodle.AddUser(account.username + "wg", account.username, account.email, account.username, account.password)
			log(err)
			if config.CreateGiteaAccount {
				opt := gitea.CreateUserOption{
					Email:      account.email,
					Username:   account.username,
					SourceID:   0,
					Password:   account.password,
					SendNotify: false,
				}
				_, _, err = giteaClient.AdminCreateUser(opt)
				log(err)
			}
		}

		submitReturn: runTemplate(w, submitTmpl, submitStruct)
	}
func getRbuMember(user string, tag string) (*discordgo.Member, bool) {
	allUsers, err := discord.GuildMembers(secret.DiscordServerID, "0", 1000)
	log(err)
	for _, element := range allUsers {
		if element.User.Username == user && element.User.Discriminator == tag{
			return element, true
		}
	}
	return nil, false
}
func UsernameExistsInMem(username string) bool {
	for key := range cacheAccounts.Iter() {
		var accInter interface{}
		accInter, _ = cacheAccounts.Get(key)
		var account account = accInter.(account)
		if account.username == username {
			return true
		}
	}
	return false
}

func discordUsernameExistsInMem(id string) bool {
	for key := range cacheAccounts.Iter() {
		var accInter interface{}
		accInter, _ = cacheAccounts.Get(key)
		var account account = accInter.(account)
		if account.discordId == id {
			return true
		}
	}
	return false
}
