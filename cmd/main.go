package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Person struct {
	ID      primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name    string             `json:"name"`
	Age     int                `json:"age"`
	Address string             `json:"address"`
}

const (
	Database   = "testdb"
	Collection = "people"
)

var client *mongo.Client

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	uri := os.Getenv("URI")
	if uri == "" {
		log.Fatal("MONGODB_URI environment variable is not set")
	}
	var err error
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	fmt.Println(os.Getenv("URI"))
	opts := options.Client().ApplyURI(os.Getenv("URI")).SetServerAPIOptions(serverAPI)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err = mongo.Connect(context.TODO(), opts)
	if err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Error pinging MongoDB:", err)
	}

	log.Println("Connected to MongoDB")
}

func main() {
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Fatal("Error disconnecting from MongoDB:", err)
		}
	}()

	router := mux.NewRouter()

	router.HandleFunc("/people", GetPeople).Methods("GET")
	router.HandleFunc("/people/{id}", GetPerson).Methods("GET")
	router.HandleFunc("/people", CreatePerson).Methods("POST")
	router.HandleFunc("/people/{id}", UpdatePerson).Methods("PUT")
	router.HandleFunc("/people/{id}", DeletePerson).Methods("DELETE")
	log.Println("Server Started")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func GetPeople(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling GET request for /people")
	collection := client.Database(Database).Collection(Collection)
	cur, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		handleError(w, err)
		return
	}
	defer cur.Close(context.Background())

	var people []Person
	for cur.Next(context.Background()) {
		var person Person
		if err := cur.Decode(&person); err != nil {
			handleError(w, err)
			return
		}
		people = append(people, person)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(people)
}

func GetPerson(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling GET request for /people/id")
	params := mux.Vars(r)
	id := params["id"]

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		handleError(w, err)
		return
	}

	collection := client.Database(Database).Collection(Collection)
	result := collection.FindOne(context.Background(), bson.M{"_id": objectID})

	var person Person
	err = result.Decode(&person)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(person)
}

func CreatePerson(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling POST request CreatePErson")
	var person Person
	err := json.NewDecoder(r.Body).Decode(&person)
	if err != nil {
		handleError(w, err)
		return
	}

	collection := client.Database(Database).Collection(Collection)
	result, err := collection.InsertOne(context.Background(), person)
	if err != nil {
		handleError(w, err)
		return
	}

	person.ID = result.InsertedID.(primitive.ObjectID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(person)
}

func UpdatePerson(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	var person Person
	err := json.NewDecoder(r.Body).Decode(&person)
	if err != nil {
		handleError(w, err)
		return
	}

	collection := client.Database(Database).Collection(Collection)
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": person})
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(person)
}

func DeletePerson(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	collection := client.Database(Database).Collection(Collection)
	_, err := collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleError(w http.ResponseWriter, err error) {
	log.Println("Error:", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
