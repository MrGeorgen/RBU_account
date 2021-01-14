package main
import (
	"golang.org/x/crypto/argon2"
	"net/http"
	"html/template"
)

func log(err error)  {
	if err!=nil {
		panic(err)
	}
}

func hashFunc(password []byte, salt []byte) []byte {
	return argon2.IDKey(password, salt, 1, 64*1024, 4, 32)
}
func runTemplate(w http.ResponseWriter, template *template.Template, templateData interface{}) {
	w.Header().Set("Content-Type", "text/html")
	var err error = template.Execute(w, templateData)
	log(err)
}
