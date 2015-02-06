package repos

import (
	"github.com/nerdyworm/sess/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	Accounts AccountsRepo
)

type AccountsRepo interface {
	FindByID(string) (*models.Account, error)
}

type mongoAccountsRepo struct {
	session  *mgo.Session
	db       *mgo.Database
	accounts *mgo.Collection
}

func NewMongoAccountsRepo(session *mgo.Session, db *mgo.Database) *mongoAccountsRepo {
	domainsDb := session.DB("domains")
	return &mongoAccountsRepo{session, domainsDb, domainsDb.C("domains")}
}

func (repo mongoAccountsRepo) FindByID(id string) (*models.Account, error) {
	account := mongoAccount{}

	err := repo.accounts.Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(&account)
	if err != nil {
		return nil, err
	}

	return &models.Account{
		ID:           account.Id.Hex(),
		InternalName: account.DomainName,
		Settings: models.AccountSettings{
			LogoPosition: account.Settings.BrandingLogoAttachmentCorner,
		},
	}, nil
}

type mongoAccountSettings struct {
	BrandingLogoAttachmentCorner string `bson:"branding_logo_attachment_corner"`
}

type mongoAccount struct {
	Id         bson.ObjectId        `bson:"_id"`
	DomainName string               `bson:"domain_name"`
	Settings   mongoAccountSettings `bson:"domain_setting"`
}
