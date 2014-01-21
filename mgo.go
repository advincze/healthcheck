package main

import "labix.org/v2/mgo"

var session *mgo.Session
var db *mgo.Database

func init() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	db = session.DB("healthcheck")
}
