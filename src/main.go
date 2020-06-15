package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/dlclark/regexp2"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)
var discord *discordgo.Session
var secret secrets_json
var cacheAccounts []account
type account struct {
	email    string
	username string
	password string
	discordUsername string
	token string
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
}

func main() {
	cacheAccounts = make([]account, 1)
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
	discord, err = discordgo.New("Bot " + secret.DiscordToken)
	log(err)
	err = discord.Open()
	log(err)
	db, err := sql.Open("mysql", secret.MysqlIndentify)
	log(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS account(" +
		"username varchar(40) NOT NULL, " +
		"email varchar(255) NOT NULL, " +
		"password varchar(255) NOT NULL, " +
		"discordUsername varchar(32) NOT NULL, " +
		"PRIMARY KEY ( username )" +
		");")
	log(err)
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
				newAccount.token, err = GenerateRandomStringURLSafe(64)
				log(err)
				dmChannel, err = discord.UserChannelCreate(newRbuMember.User.ID)
				log(err)
				discord.ChannelMessageSend(dmChannel.ID, "Bitte klicke auf den Link, um die Erstellung des Accounts abzuschließen.\nhttp://localhost:8080/submit?token="+newAccount.token)
				cacheAccounts = append(cacheAccounts, newAccount)
			}
		}
		tmpl.Execute(w, registerstruct)
		fmt.Println(registerstruct)
	})
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		token := r.FormValue("token")
		SubmitStruct.Success = false
		for i, element := range cacheAccounts {
			if element.token==token {
				fmt.Println("token")
				SubmitStruct.Success = true
				db.Exec("INSERT INTO account(username, email, password, discordUsername)" +
					"VALUES(\"" + element.username + "\", \"" + element.email + "\", \"" + element.password + "\", \"" + element.discordUsername + "\");")
				cacheAccounts = append(cacheAccounts[:i], cacheAccounts[i+1:]...) //delete element
				break
			}
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
	for _, element := range cacheAccounts {
		if element.username == username {
			return true
		}
	}
	return false
}

func discordUsernameExistsInMem(discordUsername string) bool {
	for _, element := range cacheAccounts {
		if element.discordUsername == discordUsername {
			return true
		}
	}
	return false
}