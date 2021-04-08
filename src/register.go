package main
import (
	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"strings"
	"regexp"
	"github.com/dlclark/regexp2"
	"code.gitea.io/sdk/gitea"
	"crypto/rand"
	"sync"
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
type registerStruct struct {
	Success bool
	WrongAccount WrongAccount
	AlreadyEsitsInDatabase struct{
		Username        bool
		DiscordUsername bool
	}
}
type submitStruct struct {
	Success bool
}
var accountByToken sync.Map
var usernameExitsMap sync.Map
var discordUserExitsMap sync.Map
var rusername *regexp.Regexp
var remail *regexp2.Regexp
var rpassword *regexp2.Regexp
func register(w http.ResponseWriter, r *http.Request) {
	registerStruct := registerStruct{}
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
		registerStruct.WrongAccount.Email, _ = remail.MatchString(newAccount.email)
		registerStruct.WrongAccount.Email = !registerStruct.WrongAccount.Email
		registerStruct.WrongAccount.User = !rusername.MatchString(newAccount.username)
		registerStruct.WrongAccount.Pass, _ = rpassword.MatchString(newAccount.password)
		registerStruct.WrongAccount.Pass = !registerStruct.WrongAccount.Pass
		newRbuMember, registerStruct.WrongAccount.DiscordUser = getRbuMember(newAccount.discordUsername, newAccount.discordTag)
		registerStruct.WrongAccount.DiscordUser = !registerStruct.WrongAccount.DiscordUser
		if registerStruct.WrongAccount.DiscordUser {
			goto registerReturn
		}
		newAccount.discordId = newRbuMember.User.ID
		{
			var username string
			_, usernameExitsInMem := usernameExitsMap.Load(newAccount.username)
			registerStruct.AlreadyEsitsInDatabase.Username = db.QueryRow("SELECT username FROM account WHERE username = ?", newAccount.username).Scan(&username) == nil || usernameExitsInMem
			_, discordUserExitsInMem := discordUserExitsMap.Load(newAccount.discordId)
			registerStruct.AlreadyEsitsInDatabase.DiscordUsername = db.QueryRow("SELECT username FROM account WHERE discordUserId = ?", newAccount.discordId).Scan(&username) == nil || discordUserExitsInMem
		}
		registerStruct.Success = !registerStruct.WrongAccount.User && !registerStruct.WrongAccount.Pass && !registerStruct.WrongAccount.Email && !registerStruct.WrongAccount.DiscordUser && !registerStruct.AlreadyEsitsInDatabase.DiscordUsername && !registerStruct.AlreadyEsitsInDatabase.Username
		if !registerStruct.Success {
			goto registerReturn
		}
		token, err := GenerateRandomStringURLSafe(64)
		log(err)
		var dmChannel *discordgo.Channel
		dmChannel, err = discord.UserChannelCreate(newRbuMember.User.ID)
		log(err)
		discord.ChannelMessageSend(dmChannel.ID, "Bitte klicke auf den Link, um die Erstellung des Accounts abzuschließen.\n<" + config.RootUrl + "/submit?token=" + token + ">")
		accountByToken.Store(token, newAccount)
		usernameExitsMap.Store(newAccount.username, nil)
		discordUserExitsMap.Store(newAccount.discordId, nil)
	}
	registerReturn: runTemplate(w, registerTmpl, registerStruct)
}
	func submit(w http.ResponseWriter, r *http.Request) {
		var err error
		var submitStruct submitStruct
		token := r.FormValue("token")
		var accInter interface{}
		accInter, submitStruct.Success = accountByToken.LoadAndDelete(token)
		if !submitStruct.Success {
			goto submitReturn
		}
		{
			var account account = accInter.(account)
			usernameExitsMap.Delete(account.username)
			discordUserExitsMap.Delete(account.discordId)
			salt := make([]byte, 32)
			_, err = rand.Read(salt)
			log(err)
			hash := hashFunc([]byte(account.password), salt)
			// add user to the database
			stmtCreateAccount.Exec(account.username, account.email, hash, salt, account.discordId)
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
