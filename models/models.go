package models

import "fmt"

type Account struct {
	ID           string
	InternalName string
	Settings     AccountSettings
}

type AccountSettings struct {
	LogoPosition string
}

// XXX - branding logos need to be migrated to
// {account.ID}/logo
func (a Account) LogoKey() string {
	return fmt.Sprintf("%s_branding/gui_logo", a.InternalName)
}

type User struct {
	ID         string
	Email      string
	AccountIds []string
}

type Study struct {
	ID        string
	AccountID string
}

type Instance struct {
	ID             string
	AccountID      string
	SOPInstanceUID string
}

func (i Instance) Key() string {
	return i.SOPInstanceUID
}

func IsUserInAccount(user *User, accountId string) bool {
	for _, id := range user.AccountIds {
		if id == accountId {
			return true
		}
	}

	return false
}

type Job struct {
	Name     string `json:"name"`
	Payload  []byte `json:"payload"`
	ReplyTo  string `json:"reply_to"`
	Tries    int
	Errors   []error
	tryAgain bool
}

func (job *Job) Retry() {
	job.tryAgain = true
}

func (job *Job) ShouldRetry() bool {
	return job.tryAgain && job.Tries < 2
}

func (job *Job) IncrementTries() {
	job.Tries = job.Tries + 1
}

func (job *Job) AddError(err error) {
	if job.Errors == nil {
		job.Errors = make([]error, 0)
	}

	job.Errors = append(job.Errors, err)
}
