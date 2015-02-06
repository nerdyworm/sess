package repos

import (
	"github.com/nerdyworm/sess/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	Instances InstancesRepo
)

type InstancesRepo interface {
	FindByID(string) (*models.Instance, error)
}

type mongoInstancesRepo struct {
	session   *mgo.Session
	db        *mgo.Database
	instances *mgo.Collection
}

func NewMongoInstancesRepo(session *mgo.Session, db *mgo.Database) *mongoInstancesRepo {
	return &mongoInstancesRepo{session, db, db.C("dicom_instances")}
}

func (repo mongoInstancesRepo) FindByID(id string) (*models.Instance, error) {
	instance := mongoInstance{}

	err := repo.instances.Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(&instance)
	if err != nil {
		return nil, err
	}

	return &models.Instance{
		ID:             instance.Id.Hex(),
		AccountID:      instance.DomainID.Hex(),
		SOPInstanceUID: instance.SOPInstanceUID,
	}, nil
}

type mongoInstance struct {
	Id             bson.ObjectId `bson:"_id"`
	DomainID       bson.ObjectId `bson:"domain_id"`
	SOPInstanceUID string        `bson:"sop_instance_uid"`
}
