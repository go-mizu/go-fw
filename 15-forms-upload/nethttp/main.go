package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /login", login)
	mux.HandleFunc("POST /upload", upload)

	http.ListenAndServe(":8080", mux)
}

func login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	user := r.FormValue("user")
	pass := r.FormValue("pass")

	fmt.Fprintf(w, "user=%s pass=%s\n", user, pass)
}

func upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "bad multipart", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file missing", http.StatusBadRequest)
		return
	}
	defer file.Close()

	out, err := os.Create("./" + header.Filename)
	if err != nil {
		http.Error(w, "cannot save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	io.Copy(out, file)

	fmt.Fprintf(w, "uploaded %s\n", header.Filename)
}
