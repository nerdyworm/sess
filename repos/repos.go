package repos

import (
	"log"

	"gopkg.in/mgo.v2"
)

var (
	session *mgo.Session
	db      *mgo.Database
)

func Setup() {
	var err error

	session, err = mgo.Dial("localhost:27017")
	if err != nil {
		log.Fatal(err)
	}
	db := session.DB("curiecloud_development")

	Accounts = NewMongoAccountsRepo(session, db)
	Users = NewMongoUsersRepo(session, db)
	Studies = NewMongoStudiesRepo(session, db)
	Instances = NewMongoInstancesRepo(session, db)
}

func Shutdown() {
	session.Close()
}
