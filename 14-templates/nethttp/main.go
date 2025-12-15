package main

import (
	"html/template"
	"net/http"
)

var tpl = template.Must(template.New("page").Parse(`
<!doctype html>
<html>
<head><title>{{.Title}}</title></head>
<body>
<h1>{{.Message}}</h1>
</body>
</html>
`))

type Data struct {
	Title   string
	Message string
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.Execute(w, Data{
			Title:   "Home",
			Message: "hello from template",
		}); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})

	http.ListenAndServe(":8080", mux)
}
