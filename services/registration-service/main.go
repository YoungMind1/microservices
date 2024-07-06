package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client

type Registration struct {
	ID       primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	UserID   string             `json:"user_id" bson:"user_id"`
	EventID  string             `json:"event_id" bson:"event_id"`
	Canceled bool               `json:"canceled" bson:"canceled"`
}

func store(w http.ResponseWriter, r *http.Request) {
	var registration Registration
	if err := json.NewDecoder(r.Body).Decode(&registration); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if registration.UserID == "" || registration.EventID == "" {
		http.Error(w, "user_id and event_id are required", http.StatusBadRequest)
		return
	}

	resp, err := http.Get("http://nginx/api/users/" + url.QueryEscape(registration.UserID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "This user_id does not exist", http.StatusBadRequest)
		return
	}

	resp, err = http.Get("http://nginx/api/events/" + url.QueryEscape(registration.EventID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "This event_id does not exist", http.StatusBadRequest)
		return
	}

	collection := client.Database("registrationdb").Collection("registrations")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = collection.FindOne(ctx, bson.M{"user_id": registration.UserID, "event_id": registration.EventID, "canceled": false}).Err()
	if err != mongo.ErrNoDocuments {
		http.Error(w, "This user has already registered to this event", http.StatusBadRequest)
		return
	}

	registration.Canceled = false
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	result, err := collection.InsertOne(ctx, registration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	registration.ID = result.InsertedID.(primitive.ObjectID)
	json.NewEncoder(w).Encode(registration)
}

func cancelRegistration(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	collection := client.Database("registrationdb").Collection("registrations")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = collection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": bson.M{"canceled": true}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func show(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var registration Registration
	collection := client.Database("registrationdb").Collection("registrations")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&registration)
	if err != nil {
		http.Error(w, "Registration not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(registration)
}

func main() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, _ = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://registration-mongodb:27017"))

	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	// Check the connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/create", store).Methods("POST")
	r.HandleFunc("/{id}/cancel", cancelRegistration).Methods("PUT")
	r.HandleFunc("/{id}", show).Methods("GET")
	log.Fatal(http.ListenAndServe(":80", r))
}
