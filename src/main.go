package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"github.com/bwmarrin/discordgo"
	"github.com/dlclark/regexp2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/zaddok/moodle"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"golang.org/x/crypto/argon2"
	"context"
	"time"
	"github.com/cornelk/hashmap"
	"code.gitea.io/sdk/gitea"
)
var discord *discordgo.Session
var secret secrets_json
var cacheAccounts hashmap.HashMap
type account struct {
	email    string
	username string
	password string
	discordUsername string
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
type secrets_json struct {
	DiscordToken    string `json:"discordToken"`
	MysqlIndentify  string `json:"mysqlIndentify"`
	DiscordServerID string `json:"discordServerID"`
	MoodleToken string `json:"moodleToken"`
	GiteaToken string `json:"giteaToken"`
}

func main() {
	var newRbuMember *discordgo.Member
	var dmChannel *discordgo.Channel
	var err error
	var SubmitStruct SubmitStruct
	var jsonfile *os.File
	jsonfile, err = os.Open("secrets.json")
	log(err)
	var jsondata []byte
	jsondata, err = ioutil.ReadAll(jsonfile)
	log(err)
	err = json.Unmarshal(jsondata, &secret)
	log(err)
	jsonfile.Close()
	discordgo.MakeIntent(discordgo.IntentsAll)
	discord, err = discordgo.New("Bot " + secret.DiscordToken)
	log(err)
	err = discord.Open()
	log(err)
	db, err := sql.Open("mysql", secret.MysqlIndentify)
	log(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS account(" +
		"username varchar(40) NOT NULL, " +
		"email varchar(255) NOT NULL, " +
		"hash TINYBLOB NOT NULL, " +
		"salt TINYBLOB NOT NULL, " +
		"discordUsername varchar(32) NOT NULL, " +
		"PRIMARY KEY ( username )" +
		");")
	log(err)
	giteaClient, err := gitea.NewClient("https://git.redstoneunion.de", gitea.SetToken(secret.GiteaToken))
	log(err)
	moodle := moodle.NewMoodleApi("https://exam.redstoneunion.de/", secret.MoodleToken)
	_ = moodle
	tmpl := template.Must(template.ParseFiles("tmpl/register.html"))
	submitTmpl := template.Must(template.ParseFiles("tmpl/submit.html"))
	remail := regexp2.MustCompile("^(?=.{0,255}$)(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|\"(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21\\x23-\\x5b\\x5d-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21-\\x5a\\x53-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])+)\\])$", 0)
	rusername := regexp.MustCompile("^([[:lower:]]|\\d|_|-|\\.){1,40}$")
	rpassword := regexp2.MustCompile("^(?=.{8,255}$)(?=.*[a-z])(?=.*[A-Z])(?=.*[0-9])(?=.*\\W).*$", 0)
	registerstruct := registertmpl{}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			newAccount := account{
				email:    r.FormValue("email"),
				username: r.FormValue("username"),
				password: r.FormValue("password"),
				discordUsername: r.FormValue("discordUser"),
			}
			registerstruct.WrongAccount.Email, _ = remail.MatchString(newAccount.email)
			registerstruct.WrongAccount.Email = !registerstruct.WrongAccount.Email
			registerstruct.WrongAccount.User = !rusername.MatchString(newAccount.username) || strings.Contains(newAccount.username, "\"")
			registerstruct.WrongAccount.Pass, _ = rpassword.MatchString(newAccount.password)
			registerstruct.WrongAccount.Pass = !registerstruct.WrongAccount.Pass
			newRbuMember, registerstruct.WrongAccount.DiscordUser = getRbuMember(newAccount.discordUsername)
			registerstruct.WrongAccount.DiscordUser = !registerstruct.WrongAccount.DiscordUser
			{
				var username string
				registerstruct.AlreadyEsitsInDatabase.Username = db.QueryRow("select username from account where username = ?", newAccount.username).Scan(&username) == nil || UsernameExistsInMem(newAccount.username) // check if username exits
				registerstruct.AlreadyEsitsInDatabase.DiscordUsername = db.QueryRow("select username from account where discordUsername = ?", newAccount.discordUsername).Scan(&username) == nil || discordUsernameExistsInMem(newAccount.discordUsername)
			}
			registerstruct.Success = !registerstruct.WrongAccount.User && !registerstruct.WrongAccount.Pass && !registerstruct.WrongAccount.Email && !registerstruct.WrongAccount.DiscordUser && !registerstruct.AlreadyEsitsInDatabase.DiscordUsername && !registerstruct.AlreadyEsitsInDatabase.Username
			if registerstruct.Success {
				token, err := GenerateRandomStringURLSafe(64)
				log(err)
				dmChannel, err = discord.UserChannelCreate(newRbuMember.User.ID)
				log(err)
				discord.ChannelMessageSend(dmChannel.ID, "Bitte klicke auf den Link, um die Erstellung des Accounts abzuschließen.\nhttp://localhost:8080/submit?token=" + token)
				cacheAccounts.Set(token, newAccount)
			}
		}
		tmpl.Execute(w, registerstruct)
	})
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		token := r.FormValue("token")
		var accInter interface{}
		accInter, SubmitStruct.Success = cacheAccounts.GetStringKey(token)
		var account account = accInter.(account)
		if SubmitStruct.Success {
			cacheAccounts.Del(token)
			salt := make([]byte, 32)
			_, err := rand.Read(salt)
			log(err)
			hash := argon2.IDKey([]byte(account.password), salt, 1, 64*1024, 4, 32)
			// add user to the database
			query := "INSERT INTO account(username, email, hash, salt, discordUsername) VALUES (?, ?, ?, ?, ?)"
			ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelfunc()
			stmt, err := db.PrepareContext(ctx, query)
			log(err)
			defer stmt.Close()
			_, err = stmt.ExecContext(ctx, account.username, account.email, hash, salt, account.discordUsername)
			log(err)
			//_, err = moodle.AddUser(account.username + "wg", account.username, account.email, account.username, account.password)
			log(err)
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
		err = submitTmpl.Execute(w, SubmitStruct)
		log(err)
	})

	http.ListenAndServe(":8080", nil)
}
func getRbuMember(user string) (*discordgo.Member, bool) {
	allUsers, err := discord.GuildMembers(secret.DiscordServerID, "0", 1000)
	log(err)
	for _, element := range allUsers {
		if element.User.Username==user {
			return element, true
		}
	}
	return nil, false
}
func log(err error)  {
	if err!=nil {
		panic(err)
	}
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

func discordUsernameExistsInMem(discordUsername string) bool {
	for key := range cacheAccounts.Iter() {
		var accInter interface{}
		accInter, _ = cacheAccounts.Get(key)
		var account account = accInter.(account)
		if account.discordUsername == discordUsername {
			return true
		}
	}
	return false
}
