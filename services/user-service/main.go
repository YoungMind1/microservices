package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client

type User struct {
    ID    primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
    Name  string             `json:"name" bson:"name"`
    Email string             `json:"email" bson:"email"`
}

func store(w http.ResponseWriter, r *http.Request) {
    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    collection := client.Database("userdb").Collection("users")
    ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
    result, err := collection.InsertOne(ctx, user)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    user.ID = result.InsertedID.(primitive.ObjectID)
    json.NewEncoder(w).Encode(user)
}

func show(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]
    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    var user User
    collection := client.Database("userdb").Collection("users")
    ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
    err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(user)
}

func update(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]
    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    collection := client.Database("userdb").Collection("users")
    ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
    _, err = collection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": user})
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    user.ID = objID
    json.NewEncoder(w).Encode(user)
}

func destroy(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]
    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    collection := client.Database("userdb").Collection("users")
    ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
    _, err = collection.DeleteOne(ctx, bson.M{"_id": objID})
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func main() {
    ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
    client, _ = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://user-mongodb:27017"))

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
    r.HandleFunc("/{id}", show).Methods("GET")
    r.HandleFunc("/{id}", update).Methods("PUT")
    r.HandleFunc("/{id}", destroy).Methods("DELETE")
    log.Fatal(http.ListenAndServe(":80", r))
}

