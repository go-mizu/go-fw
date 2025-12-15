package main

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
)

var tpl = template.Must(template.New("page").Parse(`
<h1>{{.Message}}</h1>
`))

func main() {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tpl.Execute(w, map[string]string{
			"Message": "hello from chi",
		})
	})

	http.ListenAndServe(":8080", r)
}
