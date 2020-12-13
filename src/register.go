package main
import (
	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"strings"
	"regexp"
	"github.com/dlclark/regexp2"
)
var rusername *regexp.Regexp
var remail *regexp2.Regexp
var rpassword *regexp2.Regexp
func register(w http.ResponseWriter, r *http.Request) {
	var err error
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
	registerReturn: err = registerTmpl.Execute(w, registerstruct)
	log(err)
}
