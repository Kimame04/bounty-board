package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"bounty-board/backend/internal/models"
)

type LoginRequest struct {
	Username string `json:"username"`
}

// Login implements When2meet-style simple authentication:
// If the username exists, returns the user; if not, registers a new adventurer.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	username := strings.TrimSpace(req.Username)
	if len(username) < 2 || len(username) > 30 {
		respondError(w, http.StatusBadRequest, "Username must be between 2 and 30 characters")
		return
	}

	var user models.User
	err := h.DB.QueryRow(
		"SELECT id, username, xp, created_at FROM users WHERE username = ? COLLATE NOCASE",
		username,
	).Scan(&user.ID, &user.Username, &user.XP, &user.CreatedAt)

	if err == sql.ErrNoRows {
		// Register new user
		res, err := h.DB.Exec("INSERT INTO users (username, xp) VALUES (?, 0)", username)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to register user")
			return
		}
		user.ID, _ = res.LastInsertId()
		user.Username = username
		user.XP = 0
		user.ComputeLevelAndTitle()
		respondJSON(w, http.StatusCreated, user)
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error during login")
		return
	}

	user.ComputeLevelAndTitle()
	respondJSON(w, http.StatusOK, user)
}

// GetUser returns details and RPG stats for a given username
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		respondError(w, http.StatusBadRequest, "Username is required")
		return
	}

	var user models.User
	err := h.DB.QueryRow(
		"SELECT id, username, xp, created_at FROM users WHERE username = ? COLLATE NOCASE",
		username,
	).Scan(&user.ID, &user.Username, &user.XP, &user.CreatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Adventurer not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	user.ComputeLevelAndTitle()
	respondJSON(w, http.StatusOK, user)
}
