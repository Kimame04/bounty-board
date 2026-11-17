package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"bounty-board/backend/internal/models"

	"github.com/go-chi/chi/v5"
)

type CreateCommentRequest struct {
	AuthorID int64  `json:"author_id"`
	Content  string `json:"content"`
}

type UpdateCommentRequest struct {
	Content string `json:"content"`
}

// ListCommentsByPost returns all dispatches/solutions for a quest
func (h *Handler) ListCommentsByPost(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	rows, err := h.DB.Query(`
		SELECT 
			c.id, c.post_id, c.author_id, u.username, u.xp,
			c.content, c.upvotes, c.created_at, c.updated_at,
			(CASE WHEN p.accepted_comment_id = c.id THEN 1 ELSE 0 END) AS is_accepted
		FROM comments c
		JOIN users u ON c.author_id = u.id
		JOIN posts p ON c.post_id = p.id
		WHERE c.post_id = ?
		ORDER BY is_accepted DESC, c.upvotes DESC, c.created_at ASC
	`, postID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to query comments")
		return
	}
	defer rows.Close()

	comments := make([]models.Comment, 0)
	for rows.Next() {
		var comment models.Comment
		var authorXP int
		var isAcceptedInt int
		if err := rows.Scan(
			&comment.ID, &comment.PostID, &comment.AuthorID, &comment.AuthorName, &authorXP,
			&comment.Content, &comment.Upvotes, &comment.CreatedAt, &comment.UpdatedAt,
			&isAcceptedInt,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to read comment")
			return
		}
		comment.IsAccepted = (isAcceptedInt == 1)
		u := models.User{XP: authorXP}
		u.ComputeLevelAndTitle()
		comment.AuthorLevel = u.Level
		comments = append(comments, comment)
	}

	respondJSON(w, http.StatusOK, comments)
}

// CreateComment posts a solution to a quest
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "postID")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		respondError(w, http.StatusBadRequest, "Content cannot be empty")
		return
	}
	if req.AuthorID <= 0 {
		respondError(w, http.StatusBadRequest, "Valid author_id is required")
		return
	}

	// Verify post exists
	var exists int
	err = h.DB.QueryRow("SELECT 1 FROM posts WHERE id = ?", postID).Scan(&exists)
	if err != nil {
		respondError(w, http.StatusNotFound, "Quest not found")
		return
	}

	res, err := h.DB.Exec(`
		INSERT INTO comments (post_id, author_id, content)
		VALUES (?, ?, ?)
	`, postID, req.AuthorID, content)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create solution dispatch")
		return
	}

	commentID, _ := res.LastInsertId()

	var comment models.Comment
	var authorXP int
	err = h.DB.QueryRow(`
		SELECT c.id, c.post_id, c.author_id, u.username, u.xp, c.content, c.upvotes, c.created_at, c.updated_at
		FROM comments c
		JOIN users u ON c.author_id = u.id
		WHERE c.id = ?
	`, commentID).Scan(
		&comment.ID, &comment.PostID, &comment.AuthorID, &comment.AuthorName, &authorXP,
		&comment.Content, &comment.Upvotes, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch created comment")
		return
	}

	u := models.User{XP: authorXP}
	u.ComputeLevelAndTitle()
	comment.AuthorLevel = u.Level

	respondJSON(w, http.StatusCreated, comment)
}

// UpvoteComment increments upvotes for a helpful comment
func (h *Handler) UpvoteComment(w http.ResponseWriter, r *http.Request) {
	commentIDStr := chi.URLParam(r, "id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid comment ID")
		return
	}

	res, err := h.DB.Exec("UPDATE comments SET upvotes = upvotes + 1 WHERE id = ?", commentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to upvote")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		respondError(w, http.StatusNotFound, "Comment not found")
		return
	}

	var upvotes int
	_ = h.DB.QueryRow("SELECT upvotes FROM comments WHERE id = ?", commentID).Scan(&upvotes)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":      commentID,
		"upvotes": upvotes,
	})
}

// UpdateComment modifies a comment
func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	commentIDStr := chi.URLParam(r, "id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid comment ID")
		return
	}

	var req UpdateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		respondError(w, http.StatusBadRequest, "Content cannot be blank")
		return
	}

	res, err := h.DB.Exec("UPDATE comments SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", content, commentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update comment")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		respondError(w, http.StatusNotFound, "Comment not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Comment updated successfully"})
}

// DeleteComment removes a comment
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	commentIDStr := chi.URLParam(r, "id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid comment ID")
		return
	}

	res, err := h.DB.Exec("DELETE FROM comments WHERE id = ?", commentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete comment")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		respondError(w, http.StatusNotFound, "Comment not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Comment deleted successfully"})
}
