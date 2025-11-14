// Package main implements a REST service for managing user subscriptions.
//
//	@title			Subscription Service API
//	@version		1.0
//	@description	API for CRUDL operations on user subscriptions.
//	@host			localhost:8080
//	@BasePath		/
//	@schemes		http
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"subscription-service/internal/config"
	"subscription-service/internal/handler"
	"subscription-service/internal/repository"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем config.yaml")
	}

	var dsn string
	if os.Getenv("DB_HOST") != "" {
		dsn = buildDSNFromEnv()
	} else {
		cfg, err := config.LoadConfig("config/config.yaml")
		if err != nil {
			log.Fatalf("Не удалось загрузить config.yaml: %v", err)
		}
		dsn = buildDSNFromConfig(cfg.Database)
	}

	db, err := repository.NewPostgresDB(dsn)
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	subRepo := repository.NewSubscriptionRepository(db)
	subHandler := handler.NewHandler(subRepo)

	r := mux.NewRouter()
	r.Use(handler.LogRequest)

	r.HandleFunc("/subscriptions", subHandler.CreateSubscription).Methods("POST")
	r.HandleFunc("/subscriptions/total", subHandler.GetTotalCost).Methods("GET")
	r.HandleFunc("/subscriptions", subHandler.ListSubscriptions).Methods("GET")
	r.HandleFunc("/subscriptions/{id}", subHandler.GetSubscriptionByID).Methods("GET")
	r.HandleFunc("/subscriptions/{id}", subHandler.UpdateSubscription).Methods("PUT")
	r.HandleFunc("/subscriptions/{id}", subHandler.DeleteSubscription).Methods("DELETE")

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// Serve Swagger UI
	r.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", http.FileServer(http.Dir("docs/"))))

	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Сервер запущен на http://localhost%s", addr)
	log.Printf("API: http://localhost%s/subscriptions", addr)
	log.Printf("Swagger: http://localhost%s/swagger/", addr)
	log.Fatal(server.ListenAndServe())
}

func buildDSNFromEnv() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslMode := os.Getenv("DB_SSL_MODE")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslMode)
}

func buildDSNFromConfig(dbCfg config.DatabaseConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.Name, dbCfg.SSLMode)
}
