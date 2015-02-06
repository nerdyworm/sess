package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/nerdyworm/sess/conversions"
	"github.com/nerdyworm/sess/repos"
	"github.com/nerdyworm/sess/storage"
	"github.com/nerdyworm/sess/workers"
	"github.com/streadway/amqp"
)

func Setup() {
	workers.Register("InstanceToJPG", InstanceToJPGFunc)
	workers.Register("InstanceToMovie", InstanceToMovieFunc)
}

func Run() {
	n := negroni.New(
		negroni.NewRecovery(),
		negroni.NewLogger(),
	)

	r := mux.NewRouter()
	r.HandleFunc("/cdn/v1/studies/{study_id}/instances/{instance_id}.jpg", imageHandler)
	r.HandleFunc("/cdn/v1/studies/{study_id}/instances/{instance_id}.mp4", movieHandler)
	n.UseHandler(r)
	n.Run(":4000")
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: grab user id from session cookies
	//userID := "5463a558236f44d54100000a"

	//user, err := repos.Users.FindByID(userID)
	//if err != nil {
	//log.Println("[ERROR] %v", err)
	//return
	//}

	vars := mux.Vars(r)
	//studyID := vars["study_id"]
	instanceID := vars["instance_id"]

	//study, err := repos.Studies.FindByID(studyID)
	//if err != nil {
	//log.Printf("[ERROR] %v\n", err)
	//return
	//}

	instance, err := repos.Instances.FindByID(instanceID)
	if err != nil {
		log.Printf("[Instance:%s][ERROR] %v\n", instanceID, err)
		return
	}

	//log.Println(models.IsUserInAccount(user, instance.AccountID))

	sizeString := r.URL.Query().Get("size")
	size, _ := strconv.Atoi(sizeString)

	brandString := r.URL.Query().Get("brand")
	brand := brandString == "true"

	converter := conversions.InstanceToJPG{
		InstanceID: instance.ID,
		Options: conversions.Options{
			Size:  size,
			Brand: brand,
		},
	}

	key := converter.Key()

	exists, err := storage.Cache.Exists(key)
	if err != nil {
		log.Printf("[Instance:%s][ERROR] %v\n", instanceID, err)
		return
	}

	if !exists {
		b, _ := json.Marshal(converter)

		job := workers.Job{
			Name:    "InstanceToJPG",
			Payload: b,
		}

		err = job.PublishAndWait()
		if err != nil {
			log.Printf("[Instance:%s][ERROR] %v\n", instanceID, err)
			return
		}
	}

	reader, err := storage.Cache.Get(key)
	if err != nil {
		log.Printf("[Instance:%s][ERROR] %v\n", instance.ID, err)
		return
	}

	w.Header().Set("Content-Type", converter.ContentType())
	io.Copy(w, reader)
}

func movieHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceID := vars["instance_id"]

	instance, err := repos.Instances.FindByID(instanceID)
	if err != nil {
		log.Printf("[Instance:%s][ERROR] %v\n", instanceID, err)
		return
	}

	converter := conversions.InstanceToMovie{
		InstanceID: instance.ID,
		Options: conversions.Options{
			Format: "mp4",
		},
	}

	key := converter.Key()

	exists, err := storage.Cache.Exists(key)
	if err != nil {
		log.Printf("[Instance:%s][ERROR] %v\n", instanceID, err)
		return
	}

	if !exists {
		b, _ := json.Marshal(converter)

		job := workers.Job{
			Name:    "InstanceToMovie",
			Payload: b,
		}

		err = job.PublishAndWait()
		if err != nil {
			log.Printf("[Instance:%s][ERROR] %v\n", instanceID, err)
			return
		}
	}

	reader, err := storage.Cache.Get(key)
	if err != nil {
		log.Printf("[Instance:%s][ERROR] %v\n", instance.ID, err)
		return
	}

	w.Header().Set("Content-Type", converter.ContentType())
	io.Copy(w, reader)
}

func InstanceToJPGFunc(job *workers.Job, message amqp.Delivery) {
	converter := conversions.InstanceToJPG{}

	err := json.Unmarshal(job.Payload, &converter)
	if err != nil {
		job.AddError(err)
		return
	}

	reader, err := converter.Convert()
	if err != nil {
		job.AddError(err)
		return
	}

	err = storage.Cache.Put(converter.Key(), reader)
	if err != nil {
		job.AddError(err)
		return
	}

	err = job.Ack()
	if err != nil {
		job.AddError(err)
		return
	}

	job.SendReply()
}

func InstanceToMovieFunc(job *workers.Job, message amqp.Delivery) {
	converter := conversions.InstanceToMovie{}

	err := json.Unmarshal(job.Payload, &converter)
	if err != nil {
		job.AddError(err)
		return
	}

	reader, err := converter.Convert()
	if err != nil {
		job.AddError(err)
		return
	}

	err = storage.Cache.Put(converter.Key(), reader)
	if err != nil {
		job.AddError(err)
		return
	}

	err = job.Ack()
	if err != nil {
		job.AddError(err)
		return
	}

	job.SendReply()
}
