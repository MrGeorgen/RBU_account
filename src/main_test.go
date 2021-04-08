package main

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"net/url"
	"bytes"
	"strings"
	"github.com/bwmarrin/discordgo"
	"github.com/dlclark/regexp2"
	"html/template"
	"io"
)

var testPassword = "*#566jgjgJJf"
var testUsername = "ausername"
var testSession string

func TestOrder(test *testing.T) {
	isTest = true
	main()
	test.Run("register", testRegister)
	test.Run("submit", testSubmit)
	test.Run("login", testLogin)
}

func testSetUsernamePassword(form url.Values) {
	form.Set("username", testUsername)
	form.Set("password", testPassword)
}

func testForm(test *testing.T, url string, statusCode int, handler http.HandlerFunc, form url.Values) *http.Response {
	request := httptest.NewRequest("POST", url, strings.NewReader(form.Encode()))
	request.Form = form
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != statusCode {
		test.Errorf("handler returned wrong status code: got %v want %v", recorder.Code, http.StatusOK)
	}
	return recorder.Result()
}

func checkBodyByTemplate(test *testing.T, response *http.Response, template *template.Template, templateData interface{}) {
	var expectedResponse bytes.Buffer
	template.Execute(&expectedResponse, templateData)
	checkBody(test, response, expectedResponse.Bytes())
}

func checkBody(test *testing.T, response *http.Response, expectedResponse []byte) {
	responseBody, _ := io.ReadAll(response.Body)
	if !bytes.Equal(expectedResponse, responseBody) {
		test.Errorf("unexpected body:\n%v", string(responseBody))
	}
}

func testRegister(test *testing.T) {
	form := url.Values{}
	testSetUsernamePassword(form)
	form.Set("email", "jffg@fv.com")
	form.Set("discordUser", secret.DiscordTestUser)
	response := testForm(test, "/register", http.StatusOK, register, form)
	checkBodyByTemplate(test, response, registerTmpl, registerStruct{Success: true})
}

func testSubmit(test *testing.T) {
	discordClient, err := discordgo.New()
	log(err)
	err = discordClient.Login(secret.DiscordTestUserEmail, secret.DiscordTestUserPassword)
	log(err)
	err = discordClient.Open()
	log(err)
	channel, err := discordClient.UserChannelCreate(secret.DiscordBotUserId)
	log(err)
	msg, err := discordClient.ChannelMessages(channel.ID, 1, "", "", channel.LastMessageID)
	log(err)
	re := regexp2.MustCompile(`(?<=\?token\=)[^>]*`, 0)
	match, _ := re.FindStringMatch(msg[0].Content)
	if match == nil {
		test.Error("The submit link was not send")
	}
	form := url.Values{}
	form.Set("token", match.String())
	response := testForm(test, "/submit", http.StatusOK, submit, form)
	checkBodyByTemplate(test, response, submitTmpl, submitStruct{Success: true})
}

func testLogin(test *testing.T) {
	form := url.Values{}
	testSetUsernamePassword(form)
	testForm(test, "/login", http.StatusSeeOther, login, form)
}
