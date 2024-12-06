package main

import (
	"net/http"
	"strconv"
)

func (s ServerBlock) HttpListen() {
	http.HandleFunc("/register", Register)
	http.ListenAndServe(s.Host+":"+strconv.Itoa(s.HTTP), nil)

}
