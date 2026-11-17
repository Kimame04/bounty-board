package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"bounty-board/backend/internal/database"
	"bounty-board/backend/internal/handlers"
	"bounty-board/backend/internal/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "data/bounty.db"
	}

	log.Printf("Connecting to database at %s...", dbPath)
	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()
	log.Println("Database initialized and schema verified successfully.")

	h := handlers.NewHandler(db)
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Cross-Origin Resource Sharing (CORS) for React frontend
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000", "*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check route
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"service": "bounty-board-backend",
			"db":      "connected",
		})
	})

	// API Routes
	r.Route("/api", func(r chi.Router) {
		// Auth / Adventurers & Leaderboard
		r.Post("/auth/login", h.Login)
		r.Get("/users", h.GetUser)
		r.Get("/leaderboard", h.GetLeaderboard)

		// Topics (Guild Halls)
		r.Get("/topics", h.ListTopics)
		r.Post("/topics", h.CreateTopic)
		r.Get("/topics/{id}", h.GetTopic)
		r.Delete("/topics/{id}", h.DeleteTopic)

		// Posts (Quests) under Topics
		r.Get("/topics/{topicID}/posts", h.ListPostsByTopic)
		r.Post("/topics/{topicID}/posts", h.CreatePost)

		// Single Post Operations
		r.Get("/posts/{id}", h.GetPost)
		r.Put("/posts/{id}", h.UpdatePost)
		r.Delete("/posts/{id}", h.DeletePost)
		r.Post("/posts/{id}/resolve", h.ResolveQuest)

		// Comments (Solutions) under Posts
		r.Get("/posts/{postID}/comments", h.ListCommentsByPost)
		r.Post("/posts/{postID}/comments", h.CreateComment)

		// Single Comment Operations
		r.Put("/comments/{id}", h.UpdateComment)
		r.Delete("/comments/{id}", h.DeleteComment)
		r.Post("/comments/{id}/upvote", h.UpvoteComment)
	})

	// Mount embedded frontend React SPA
	r.Handle("/*", web.SPAHandler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("BountyBoard server listening on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
