package main

import (
	"log"
	"os"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Person struct {
	ID        bson.ObjectId `bson:"_id,omitempty"`
	Name      string
	Hobbies   []string
	Timestamp time.Time
}

func main() {
	mgo.SetDebug(true)
	logger := log.New(os.Stdout,
		"INFO: ",
		log.Lshortfile)
	mgo.SetLogger(logger)
	session, err := mgo.Dial("127.0.0.1:9999/test")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	// Collection People
	c := session.DB("test").C("people")

	// Insert Datas
	err = c.Insert(&Person{Name: "Ale", Hobbies: []string{"coding", "running"}})
	if err != nil {
		panic(err)
	}
}
