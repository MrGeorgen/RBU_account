package main
import (
	"golang.org/x/crypto/argon2"
	"net/http"
	"html/template"
	"github.com/gorilla/csrf"
	"github.com/mitchellh/mapstructure"
)

func log(err error)  {
	if err!=nil {
		panic(err)
	}
}

func hashFunc(password []byte, salt []byte) []byte {
	return argon2.IDKey(password, salt, 1, 64*1024, 4, 32)
}
func runTemplate(r *http.Request, w http.ResponseWriter, template *template.Template, templateData interface{}) {
	var templateMap map[string]interface{}
	mapstructure.Decode(templateData, &templateMap)
	templateMap[csrf.TemplateTag] = csrf.TemplateField(r)
	w.Header().Set("Content-Type", "text/html")
	var err error = template.Execute(w, templateMap)
	log(err)
}
