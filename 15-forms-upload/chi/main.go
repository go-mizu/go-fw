package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	r.Post("/login", login)
	r.Post("/upload", upload)

	http.ListenAndServe(":8080", r)
}

func login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Fprintf(w, "user=%s\n", r.FormValue("user"))
}

func upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	defer r.MultipartForm.RemoveAll()

	file, header, _ := r.FormFile("file")
	defer file.Close()

	out, _ := os.Create(header.Filename)
	defer out.Close()

	io.Copy(out, file)

	fmt.Fprintf(w, "uploaded %s\n", header.Filename)
}
