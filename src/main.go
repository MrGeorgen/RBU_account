package main

import (
	"net/http"
	"database/sql"
	"encoding/json"
	"github.com/bwmarrin/discordgo"
	"github.com/dlclark/regexp2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/zaddok/moodle"
	"html/template"
	"io/ioutil"
	"os"
	"regexp"
	"code.gitea.io/sdk/gitea"
	"fmt"
)
var discord *discordgo.Session
var secret secrets_json
var config config_json
var db *sql.DB
var giteaClient *gitea.Client
var registerTmpl *template.Template
var submitTmpl *template.Template
var loginTmpl *template.Template
type secrets_json struct {
	DiscordToken    string `json:"discordToken"`
	MysqlIndentify  string `json:"mysqlIndentify"`
	DiscordServerID string `json:"discordServerID"`
	MoodleToken string `json:"moodleToken"`
	GiteaToken string `json:"giteaToken"`
	ApiToken string `json:"apiToken"`
}
type config_json struct {
	CreateGiteaAccount bool `json:"createGiteaAccount"`
	Port uint16 `json:"port"`
	RootUrl string `json:"rootUrl"`
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
	log(err)
	jsonfile.Close()
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
	registerTmpl = template.Must(template.ParseFiles("tmpl/register.html"))
	submitTmpl = template.Must(template.ParseFiles("tmpl/submit.html"))
	loginTmpl = template.Must(template.ParseFiles("tmpl/login.html"))
	remail = regexp2.MustCompile("^(?=.{0,255}$)(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|\"(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21\\x23-\\x5b\\x5d-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21-\\x5a\\x53-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])+)\\])$", 0)
	rusername = regexp.MustCompile("^([[:lower:]]|\\d|_|-|\\.){1,40}$")
	rpassword = regexp2.MustCompile("^(?=.{8,255}$)(?=.*[a-z])(?=.*[A-Z])(?=.*[0-9])(?=.*\\W).*$", 0)
	http.HandleFunc("/register", register)
	http.HandleFunc("/submit", submit)
	http.HandleFunc("/login", login)
	http.HandleFunc("/api/accountinfo", accountApi)

	http.ListenAndServe(":" + fmt.Sprint(config.Port), nil)
}
