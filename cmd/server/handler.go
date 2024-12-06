package main

import "net/http"



func Register(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}
