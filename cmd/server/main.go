package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v4"
	"github.com/go-chi/chi/v4/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openclaw/notification-service/internal/database"
	"github.com/openclaw/notification-service/internal/models"
	"github.com/openclaw/notification-service/pkg/telemetry"
)

type application struct {
	db *pgxpool.Pool
}

func main() {
	tp, err := telemetry.InitTracer()
	if err != nil {
		log.Fatal("Could not initialize tracer:", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	dbpool, err := database.NewDBConnection()
	if err != nil {
		log.Fatal("Could not connect to the database:", err)
	}
	defer dbpool.Close()

	app := &application{
		db: dbpool,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/health", app.healthCheckHandler)
	r.Post("/notifications", app.createNotificationHandler)
	r.Get("/notifications/{user_id}", app.getNotificationsByUserHandler)
	r.Patch("/notifications/{id}/read", app.markNotificationAsReadHandler)

	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", r)
}

func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if err := app.db.Ping(context.Background()); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("OK"))
}

func (app *application) createNotificationHandler(w http.ResponseWriter, r *http.Request) {
	var notification models.Notification
	err := json.NewDecoder(r.Body).Decode(&notification)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO notifications (user_id, type, title, message, created_at)
			  VALUES ($1, $2, $3, $4, NOW())
			  RETURNING id, created_at`

	err = app.db.QueryRow(context.Background(), query, notification.UserID, notification.Type, notification.Title, notification.Message).Scan(&notification.ID, &notification.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to create notification", http.StatusInternalServerError)
		log.Println("Failed to create notification:", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(notification)
}

func (app *application) getNotificationsByUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "user_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	query := `SELECT id, user_id, type, title, message, read_at, sent_at, created_at
			  FROM notifications
			  WHERE user_id = $1
			  ORDER BY created_at DESC`

	rows, err := app.db.Query(context.Background(), query, userID)
	if err != nil {
		http.Error(w, "Failed to retrieve notifications", http.StatusInternalServerError)
		log.Println("Failed to retrieve notifications:", err)
		return
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var notification models.Notification
		err := rows.Scan(&notification.ID, &notification.UserID, &notification.Type, &notification.Title, &notification.Message, &notification.ReadAt, &notification.SentAt, &notification.CreatedAt)
		if err != nil {
			http.Error(w, "Failed to scan notification", http.StatusInternalServerError)
			log.Println("Failed to scan notification:", err)
			return
		}
		notifications = append(notifications, notification)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

func (app *application) markNotificationAsReadHandler(w http.ResponseWriter, r *http.Request) {
	notificationID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid notification ID", http.StatusBadRequest)
		return
	}

	query := `UPDATE notifications
			  SET read_at = NOW()
			  WHERE id = $1`

	_, err = app.db.Exec(context.Background(), query, notificationID)
	if err != nil {
		http.Error(w, "Failed to mark notification as read", http.StatusInternalServerError)
		log.Println("Failed to mark notification as read:", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
