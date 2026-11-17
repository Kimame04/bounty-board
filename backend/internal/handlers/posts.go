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

type CreatePostRequest struct {
	AuthorID int64  `json:"author_id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

type UpdatePostRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ListPostsByTopic retrieves quests for a specific topic, optionally filtered by status
func (h *Handler) ListPostsByTopic(w http.ResponseWriter, r *http.Request) {
	topicIDStr := chi.URLParam(r, "topicID")
	topicID, err := strconv.ParseInt(topicIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid topic ID")
		return
	}

	statusFilter := r.URL.Query().Get("status") // "open", "resolved", or empty for all

	query := `
		SELECT 
			p.id, p.topic_id, p.author_id, u.username, u.xp,
			p.title, p.content, p.is_resolved, p.accepted_comment_id,
			p.created_at, p.updated_at,
			COUNT(c.id) AS comment_count
		FROM posts p
		JOIN users u ON p.author_id = u.id
		LEFT JOIN comments c ON c.post_id = p.id
		WHERE p.topic_id = ?
	`
	args := []interface{}{topicID}

	if statusFilter == "open" {
		query += " AND p.is_resolved = 0"
	} else if statusFilter == "resolved" {
		query += " AND p.is_resolved = 1"
	}

	query += " GROUP BY p.id ORDER BY p.created_at DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to query quests")
		return
	}
	defer rows.Close()

	posts := make([]models.Post, 0)
	for rows.Next() {
		var post models.Post
		var authorXP int
		if err := rows.Scan(
			&post.ID, &post.TopicID, &post.AuthorID, &post.AuthorName, &authorXP,
			&post.Title, &post.Content, &post.IsResolved, &post.AcceptedCommentID,
			&post.CreatedAt, &post.UpdatedAt, &post.CommentCount,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to read post row")
			return
		}
		u := models.User{XP: authorXP}
		u.ComputeLevelAndTitle()
		post.AuthorLevel = u.Level
		posts = append(posts, post)
	}

	respondJSON(w, http.StatusOK, posts)
}

// CreatePost creates a new quest under a topic
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	topicIDStr := chi.URLParam(r, "topicID")
	topicID, err := strconv.ParseInt(topicIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid topic ID")
		return
	}

	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		respondError(w, http.StatusBadRequest, "Title and content cannot be blank")
		return
	}
	if req.AuthorID <= 0 {
		respondError(w, http.StatusBadRequest, "Valid author_id is required")
		return
	}

	// Verify topic exists
	var exists int
	err = h.DB.QueryRow("SELECT 1 FROM topics WHERE id = ?", topicID).Scan(&exists)
	if err != nil {
		respondError(w, http.StatusNotFound, "Guild hall not found")
		return
	}

	// Insert post
	res, err := h.DB.Exec(`
		INSERT INTO posts (topic_id, author_id, title, content)
		VALUES (?, ?, ?, ?)
	`, topicID, req.AuthorID, title, content)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create quest")
		return
	}

	postID, _ := res.LastInsertId()
	h.fetchAndRespondPost(w, postID, http.StatusCreated)
}

// GetPost returns a single post by ID
func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}
	h.fetchAndRespondPost(w, postID, http.StatusOK)
}

func (h *Handler) fetchAndRespondPost(w http.ResponseWriter, postID int64, statusCode int) {
	var post models.Post
	var authorXP int

	err := h.DB.QueryRow(`
		SELECT 
			p.id, p.topic_id, p.author_id, u.username, u.xp,
			p.title, p.content, p.is_resolved, p.accepted_comment_id,
			p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM comments WHERE post_id = p.id) AS comment_count
		FROM posts p
		JOIN users u ON p.author_id = u.id
		WHERE p.id = ?
	`, postID).Scan(
		&post.ID, &post.TopicID, &post.AuthorID, &post.AuthorName, &authorXP,
		&post.Title, &post.Content, &post.IsResolved, &post.AcceptedCommentID,
		&post.CreatedAt, &post.UpdatedAt, &post.CommentCount,
	)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Quest not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch quest")
		return
	}

	u := models.User{XP: authorXP}
	u.ComputeLevelAndTitle()
	post.AuthorLevel = u.Level

	respondJSON(w, statusCode, post)
}

// UpdatePost edits an existing post
func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	var req UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		respondError(w, http.StatusBadRequest, "Title and content cannot be blank")
		return
	}

	res, err := h.DB.Exec(`
		UPDATE posts SET title = ?, content = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, title, content, postID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update quest")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		respondError(w, http.StatusNotFound, "Quest not found")
		return
	}

	h.fetchAndRespondPost(w, postID, http.StatusOK)
}

// DeletePost deletes a post and cascades to comments
func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	res, err := h.DB.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete quest")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		respondError(w, http.StatusNotFound, "Quest not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Quest deleted successfully"})
}
