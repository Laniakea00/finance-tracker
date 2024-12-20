package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RequestData struct {
	Message string `json:"message"`
}

type ResponseData struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Message struct {
	ID      primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Message string             `json:"message" bson:"message"`
}

var client *mongo.Client
var collection *mongo.Collection

func connectToMongoDB() (*mongo.Client, error) {
	uri := "mongodb+srv://berikzhanalan:123@cluster0.0m4fx.mongodb.net/?retryWrites=true&w=majority"
	clientOptions := options.Client().ApplyURI(uri)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	fmt.Println("Успешное подключение к MongoDB")
	return client, nil
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		var messages []Message
		cursor, err := collection.Find(context.Background(), bson.D{})
		if err != nil {
			http.Error(w, "Ошибка при извлечении данных", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(context.Background())

		for cursor.Next(context.Background()) {
			var msg Message
			if err := cursor.Decode(&msg); err != nil {
				http.Error(w, "Ошибка при декодировании данных", http.StatusInternalServerError)
				return
			}
			messages = append(messages, msg)
		}

		json.NewEncoder(w).Encode(messages)

	case http.MethodPost:
		var reqData RequestData
		err := json.NewDecoder(r.Body).Decode(&reqData)
		if err != nil || reqData.Message == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponseData{
				Status:  "fail",
				Message: "Некорректное JSON-сообщение",
			})
			return
		}

		msg := Message{Message: reqData.Message}
		_, err = collection.InsertOne(context.Background(), msg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ResponseData{
				Status:  "fail",
				Message: "Ошибка при сохранении данных",
			})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ResponseData{
			Status:  "success",
			Message: "Запись успешно добавлена",
		})

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponseData{
				Status:  "fail",
				Message: "ID записи не указан",
			})
			return
		}

		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ResponseData{
				Status:  "fail",
				Message: "Неверный формат ID",
			})
			return
		}

		_, err = collection.DeleteOne(context.Background(), bson.M{"_id": objID})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ResponseData{
				Status:  "fail",
				Message: "Ошибка при удалении записи",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ResponseData{
			Status:  "success",
			Message: "Запись успешно удалена",
		})
	}
}

func main() {
	var err error
	client, err = connectToMongoDB()
	if err != nil {
		log.Fatal(err)
	}
	collection = client.Database("financetracker").Collection("messages")

	http.HandleFunc("/api", handleAPI)
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	fmt.Println("Сервер запущен на http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
