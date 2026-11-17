package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"bounty-board/backend/internal/models"

	"github.com/go-chi/chi/v5"
)

type CreateTopicRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListTopics returns all Guild Halls with their active quest counts
func (h *Handler) ListTopics(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT t.id, t.name, t.description, t.created_at, COUNT(p.id) AS post_count
		FROM topics t
		LEFT JOIN posts p ON p.topic_id = t.id
		GROUP BY t.id
		ORDER BY t.name ASC
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to query guild halls")
		return
	}
	defer rows.Close()

	topics := make([]models.Topic, 0)
	for rows.Next() {
		var topic models.Topic
		if err := rows.Scan(&topic.ID, &topic.Name, &topic.Description, &topic.CreatedAt, &topic.PostCount); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to read topics")
			return
		}
		topics = append(topics, topic)
	}

	respondJSON(w, http.StatusOK, topics)
}

// CreateTopic creates a new Guild Hall (Topic)
func (h *Handler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	var req CreateTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 50 {
		respondError(w, http.StatusBadRequest, "Topic name must be between 1 and 50 characters")
		return
	}

	res, err := h.DB.Exec(
		"INSERT INTO topics (name, description) VALUES (?, ?)",
		name, strings.TrimSpace(req.Description),
	)
	if err != nil {
		respondError(w, http.StatusConflict, "A guild hall with this name already exists")
		return
	}

	id, _ := res.LastInsertId()
	topic := models.Topic{
		ID:          id,
		Name:        name,
		Description: req.Description,
		PostCount:   0,
	}
	respondJSON(w, http.StatusCreated, topic)
}

// GetTopic returns details of a single topic
func (h *Handler) GetTopic(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid topic ID")
		return
	}

	var topic models.Topic
	err = h.DB.QueryRow(`
		SELECT t.id, t.name, t.description, t.created_at, COUNT(p.id) AS post_count
		FROM topics t
		LEFT JOIN posts p ON p.topic_id = t.id
		WHERE t.id = ?
		GROUP BY t.id
	`, id).Scan(&topic.ID, &topic.Name, &topic.Description, &topic.CreatedAt, &topic.PostCount)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Guild hall not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch topic")
		return
	}

	respondJSON(w, http.StatusOK, topic)
}

// DeleteTopic deletes a topic and cascades deletion to posts and comments
func (h *Handler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid topic ID")
		return
	}

	res, err := h.DB.Exec("DELETE FROM topics WHERE id = ?", id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete topic")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		respondError(w, http.StatusNotFound, "Guild hall not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Guild hall deleted successfully"})
}
