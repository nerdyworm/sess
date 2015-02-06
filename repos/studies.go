package repos

import (
	"github.com/nerdyworm/sess/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	Studies StudiesRepo
)

type StudiesRepo interface {
	FindByID(string) (*models.Study, error)
}

type mongoStudiesRepo struct {
	session *mgo.Session
	db      *mgo.Database
	studies *mgo.Collection
}

func NewMongoStudiesRepo(session *mgo.Session, db *mgo.Database) *mongoStudiesRepo {
	return &mongoStudiesRepo{session, db, db.C("dicom_studies")}
}

func (repo mongoStudiesRepo) FindByID(id string) (*models.Study, error) {
	study := mongoStudy{}

	err := repo.studies.Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(&study)
	if err != nil {
		return nil, err
	}

	return &models.Study{
		ID: study.Id.Hex(),
	}, nil
}

type mongoStudy struct {
	Id bson.ObjectId `bson:"_id"`
}
