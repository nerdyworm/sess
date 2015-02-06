package repos

import (
	"github.com/nerdyworm/sess/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	Users UsersRepo
)

type UsersRepo interface {
	FindByID(string) (*models.User, error)
}

type mongoUserRepo struct {
	session *mgo.Session
	db      *mgo.Database
	users   *mgo.Collection
}

func NewMongoUsersRepo(session *mgo.Session, db *mgo.Database) *mongoUserRepo {
	repo := &mongoUserRepo{session, db, db.C("users")}
	return repo
}

func (repo mongoUserRepo) FindByID(id string) (*models.User, error) {
	u := mongoUser{}

	err := repo.users.Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(&u)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:    u.Id.Hex(),
		Email: u.Email,
	}

	for _, domainId := range u.DomainIds {
		user.AccountIds = append(user.AccountIds, domainId.Hex())
	}

	return user, nil
}

type mongoUser struct {
	Id        bson.ObjectId   `bson:"_id"`
	Email     string          `bson:"email"`
	DomainIds []bson.ObjectId `bson:"domain_ids"`
}
