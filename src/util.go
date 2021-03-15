package main
import (
	"golang.org/x/crypto/argon2"
	"net/http"
	"html/template"
	"strings"
	"context"
	"time"
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
func databaseInsert(query string, values ...interface{}) {
	query += " VALUES (" + strings.Repeat("?,", len(values) - 1) + "?);"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	log(err)
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, values...)
	log(err)
}
