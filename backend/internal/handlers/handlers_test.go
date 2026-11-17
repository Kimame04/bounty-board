package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"bounty-board/backend/internal/database"
	"bounty-board/backend/internal/models"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter(t *testing.T) (*chi.Mux, func()) {
	testDBPath := "test_handlers.db"
	db, err := database.InitDB(testDBPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	h := NewHandler(db)
	r := chi.NewRouter()

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login", h.Login)
		r.Get("/users", h.GetUser)
		r.Get("/leaderboard", h.GetLeaderboard)

		r.Get("/topics", h.ListTopics)
		r.Post("/topics", h.CreateTopic)
		r.Get("/topics/{id}", h.GetTopic)
		r.Delete("/topics/{id}", h.DeleteTopic)

		r.Get("/topics/{topicID}/posts", h.ListPostsByTopic)
		r.Post("/topics/{topicID}/posts", h.CreatePost)
		r.Get("/posts/{id}", h.GetPost)
		r.Put("/posts/{id}", h.UpdatePost)
		r.Delete("/posts/{id}", h.DeletePost)
		r.Post("/posts/{id}/resolve", h.ResolveQuest)

		r.Get("/posts/{postID}/comments", h.ListCommentsByPost)
		r.Post("/posts/{postID}/comments", h.CreateComment)
		r.Put("/comments/{id}", h.UpdateComment)
		r.Delete("/comments/{id}", h.DeleteComment)
		r.Post("/comments/{id}/upvote", h.UpvoteComment)
	})

	teardown := func() {
		db.Close()
		os.Remove(testDBPath)
	}

	return r, teardown
}

func TestCompleteCRUDLifecycle(t *testing.T) {
	r, teardown := setupTestRouter(t)
	defer teardown()

	// 1. Auth: Register / Login User
	loginPayload := `{"username": "Kieran"}`
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(loginPayload))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("Login failed with status %d: %s", rec.Code, rec.Body.String())
	}

	var user models.User
	if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
		t.Fatalf("Failed to parse user: %v", err)
	}
	if user.Username != "Kieran" || user.ID == 0 {
		t.Fatalf("Unexpected user: %+v", user)
	}

	// 2. Topics: Create Topic
	topicPayload := `{"name": "Go Magic", "description": "Channeling Go routines"}`
	req = httptest.NewRequest("POST", "/api/topics", bytes.NewBufferString(topicPayload))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Create topic failed with status %d: %s", rec.Code, rec.Body.String())
	}
	var topic models.Topic
	json.Unmarshal(rec.Body.Bytes(), &topic)

	// 3. Posts: Create Quest
	postPayload := `{"author_id": 1, "title": "Deadlock in Goroutines", "content": "How do channels avoid deadlocks?"}`
	req = httptest.NewRequest("POST", "/api/topics/1/posts", bytes.NewBufferString(postPayload))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Create post failed with status %d: %s", rec.Code, rec.Body.String())
	}
	var post models.Post
	json.Unmarshal(rec.Body.Bytes(), &post)

	if post.AuthorName != "Kieran" || post.CommentCount != 0 {
		t.Fatalf("Unexpected post response: %+v", post)
	}

	// 4. Comments: Add Solution
	commentPayload := `{"author_id": 1, "content": "Always ensure senders close unbuffered channels or use select."}`
	req = httptest.NewRequest("POST", "/api/posts/1/comments", bytes.NewBufferString(commentPayload))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Create comment failed with status %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Comments: Upvote
	req = httptest.NewRequest("POST", "/api/comments/1/upvote", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Upvote comment failed with status %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Comments: Verify in list
	req = httptest.NewRequest("GET", "/api/posts/1/comments", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var comments []models.Comment
	json.Unmarshal(rec.Body.Bytes(), &comments)
	if len(comments) != 1 || comments[0].Upvotes != 1 {
		t.Fatalf("Expected 1 comment with 1 upvote, got %+v", comments)
	}

	// 7. Posts: Update post
	updatePayload := `{"title": "Deadlock Solved!", "content": "Updated content."}`
	req = httptest.NewRequest("PUT", "/api/posts/1", bytes.NewBufferString(updatePayload))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Update post failed: %s", rec.Body.String())
	}

	// 8. Delete Post (Cascades)
	req = httptest.NewRequest("DELETE", "/api/posts/1", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Delete post failed: %s", rec.Body.String())
	}
}
