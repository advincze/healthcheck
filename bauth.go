package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

const (
	authuser     = "foo"
	authpassword = "bar"
)

func auth(user, pwd string) bool {
	if user == authuser && pwd == authpassword {
		return true
	}
	return false
}

func basicAuth(h http.HandlerFunc, authFn func(string, string) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// extract username and password
		authInfo := req.Header.Get("Authorization")
		if authInfo == "" {
			// No authorization info, return 401
			Unauthorized(w)
			return
		}
		parts := strings.Split(authInfo, " ")
		if len(parts) != 2 {
			BadRequest(w, "Bad authorization header")
			return
		}
		scheme := parts[0]
		creds, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			BadRequest(w, "Bad credentials encoding")
			return
		}
		index := bytes.Index(creds, []byte(":"))
		if scheme != "Basic" || index < 0 {
			BadRequest(w, "Bad authorization header")
			return
		}
		username, pwd := string(creds[:index]), string(creds[index+1:])
		if authFn(username, pwd) {
			h(w, req)
		} else {
			Unauthorized(w)
		}
	}
}

func BadRequest(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	if msg == "" {
		msg = "Bad Request"
	}
	w.Write([]byte(msg))
}

func Unauthorized(w http.ResponseWriter) {
	w.Header().Set("Www-Authenticate", fmt.Sprintf("Basic"))
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}
