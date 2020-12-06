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
var config config_json
var cacheAccounts hashmap.HashMap
var db *sql.DB
var giteaClient *gitea.Client
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
type secrets_json struct {
	DiscordToken    string `json:"discordToken"`
	MysqlIndentify  string `json:"mysqlIndentify"`
	DiscordServerID string `json:"discordServerID"`
	MoodleToken string `json:"moodleToken"`
	GiteaToken string `json:"giteaToken"`
}
type config_json struct {
	CreateGiteaAccount bool `json:"createGiteaAccount"`
}

func main() {
	var err error
	var jsonfile *os.File
	jsonfile, err = os.Open("secrets.json")
	log(err)
	var jsondata []byte
	jsondata, err = ioutil.ReadAll(jsonfile)
	log(err)
	err = json.Unmarshal(jsondata, &secret)
	log(err)
	jsonfile.Close()
	jsonfile, err = os.Open("config.json")
	log(err)
	jsondata, err = ioutil.ReadAll(jsonfile)
	log(err)
	err = json.Unmarshal(jsondata, &config)
	discordgo.MakeIntent(discordgo.IntentsAll)
	discord, err = discordgo.New("Bot " + secret.DiscordToken)
	log(err)
	err = discord.Open()
	log(err)
	db, err = sql.Open("mysql", secret.MysqlIndentify)
	log(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS account(" +
		"username varchar(40) NOT NULL, " +
		"email varchar(255) NOT NULL, " +
		"hash TINYBLOB NOT NULL, " +
		"salt TINYBLOB NOT NULL, " +
		"discordUserId varchar(32) NOT NULL, " +
		"PRIMARY KEY ( username )" +
		");")
	log(err)
	giteaClient, err = gitea.NewClient("https://git.redstoneunion.de", gitea.SetToken(secret.GiteaToken))
	log(err)
	moodle := moodle.NewMoodleApi("https://exam.redstoneunion.de/", secret.MoodleToken)
	_ = moodle
	tmpl := template.Must(template.ParseFiles("tmpl/register.html"))
	submitTmpl := template.Must(template.ParseFiles("tmpl/submit.html"))
	remail := regexp2.MustCompile("^(?=.{0,255}$)(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|\"(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21\\x23-\\x5b\\x5d-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21-\\x5a\\x53-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])+)\\])$", 0)
	rusername := regexp.MustCompile("^([[:lower:]]|\\d|_|-|\\.){1,40}$")
	rpassword := regexp2.MustCompile("^(?=.{8,255}$)(?=.*[a-z])(?=.*[A-Z])(?=.*[0-9])(?=.*\\W).*$", 0)
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
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
				registerstruct.AlreadyEsitsInDatabase.Username = db.QueryRow("select username from account where username = ?", newAccount.username).Scan(&username) == nil || UsernameExistsInMem(newAccount.username) // check if username exits
				registerstruct.AlreadyEsitsInDatabase.DiscordUsername = db.QueryRow("select username from account where discordUserId = ?", newAccount.discordId).Scan(&username) == nil || discordUsernameExistsInMem(newAccount.discordId)
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
		registerReturn: err = tmpl.Execute(w, registerstruct)
		log(err)
	})
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
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
			_, err := rand.Read(salt)
			log(err)
			hash := argon2.IDKey([]byte(account.password), salt, 1, 64*1024, 4, 32)
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

		submitReturn: err = submitTmpl.Execute(w, submitStruct)
		log(err)
	})

	http.ListenAndServe(":8080", nil)
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
