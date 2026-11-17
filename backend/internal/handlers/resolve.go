package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"bounty-board/backend/internal/models"

	"github.com/go-chi/chi/v5"
)

type ResolveRequest struct {
	RequesterID       int64  `json:"requester_id"`
	AcceptedCommentID *int64 `json:"accepted_comment_id,omitempty"`
}

type PartyRewardSummary struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	XPAwarded int    `json:"xp_awarded"`
	Reason    string `json:"reason"`
	NewXP     int    `json:"new_xp"`
	NewLevel  int    `json:"new_level"`
	NewTitle  string `json:"new_title"`
}

type ResolveResponse struct {
	Post    models.Post          `json:"post"`
	Rewards []PartyRewardSummary `json:"rewards"`
	Message string               `json:"message"`
}

// ResolveQuest marks a quest as resolved/helpful and distributes party XP within an ACID transaction
func (h *Handler) ResolveQuest(w http.ResponseWriter, r *http.Request) {
	postIDStr := chi.URLParam(r, "id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid quest ID")
		return
	}

	var req ResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.RequesterID <= 0 {
		respondError(w, http.StatusBadRequest, "Valid requester_id is required")
		return
	}

	// 1. Begin ACID Transaction
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to initiate transaction")
		return
	}
	defer tx.Rollback() // Safe rollback if not committed

	// 2. Fetch Quest details under lock
	var postAuthorID int64
	var isResolved bool
	err = tx.QueryRow(`
		SELECT author_id, is_resolved FROM posts WHERE id = ?
	`, postID).Scan(&postAuthorID, &isResolved)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Quest not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to read quest state")
		return
	}

	// 3. Authorization Check: Only quest author can resolve
	if postAuthorID != req.RequesterID {
		respondError(w, http.StatusForbidden, "Only the quest author can mark this quest resolved")
		return
	}

	// 4. Ensure quest is not already resolved
	if isResolved {
		respondError(w, http.StatusConflict, "This quest has already been resolved and rewarded")
		return
	}

	var acceptedAuthorID int64 = 0
	// 5. If an accepted solution was picked, verify it belongs to this post
	if req.AcceptedCommentID != nil && *req.AcceptedCommentID > 0 {
		err = tx.QueryRow(`
			SELECT author_id FROM comments WHERE id = ? AND post_id = ?
		`, *req.AcceptedCommentID, postID).Scan(&acceptedAuthorID)

		if err == sql.ErrNoRows {
			respondError(w, http.StatusBadRequest, "Accepted comment does not exist on this quest")
			return
		} else if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to verify accepted comment")
			return
		}
	}

	// 6. Update the Post as resolved
	_, err = tx.Exec(`
		UPDATE posts 
		SET is_resolved = 1, accepted_comment_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, req.AcceptedCommentID, postID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update quest status")
		return
	}

	rewardsSummary := make([]PartyRewardSummary, 0)

	// Helper function inside tx to award XP and track summary
	awardXP := func(userID int64, xp int, reason string) error {
		// Insert reward record
		_, err := tx.Exec(`
			INSERT INTO quest_rewards (post_id, user_id, xp_awarded, reason)
			VALUES (?, ?, ?, ?)
		`, postID, userID, xp, reason)
		if err != nil {
			return err
		}

		// Update user XP
		_, err = tx.Exec(`
			UPDATE users SET xp = xp + ? WHERE id = ?
		`, xp, userID)
		if err != nil {
			return err
		}

		// Fetch updated user info
		var username string
		var newXP int
		err = tx.QueryRow("SELECT username, xp FROM users WHERE id = ?", userID).Scan(&username, &newXP)
		if err != nil {
			return err
		}

		u := models.User{XP: newXP}
		u.ComputeLevelAndTitle()

		rewardsSummary = append(rewardsSummary, PartyRewardSummary{
			UserID:    userID,
			Username:  username,
			XPAwarded: xp,
			Reason:    reason,
			NewXP:     newXP,
			NewLevel:  u.Level,
			NewTitle:  u.Title,
		})
		return nil
	}

	// 7. Award Accepted Solution Author (+50 XP)
	if acceptedAuthorID > 0 {
		if err := awardXP(acceptedAuthorID, 50, "accepted_solution"); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to award accepted solution XP")
			return
		}
	}

	// 8. Award Party Commenters (+15 XP each)
	// Anyone who commented on this post, except the quest author and accepted author (who already got 50 XP)
	rows, err := tx.Query(`
		SELECT DISTINCT author_id FROM comments
		WHERE post_id = ? AND author_id != ? AND author_id != ?
	`, postID, postAuthorID, acceptedAuthorID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to query party contributors")
		return
	}
	defer rows.Close()

	partyIDs := make([]int64, 0)
	for rows.Next() {
		var partyUserID int64
		if err := rows.Scan(&partyUserID); err == nil {
			partyIDs = append(partyIDs, partyUserID)
		}
	}
	rows.Close()

	for _, partyUserID := range partyIDs {
		if err := awardXP(partyUserID, 15, "party_participant"); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to award party participant XP")
			return
		}
	}

	// 9. Award Quest Author (+10 XP) for completing their quest
	if err := awardXP(postAuthorID, 10, "poster_bonus"); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to award poster completion XP")
		return
	}

	// 10. Commit ACID Transaction
	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to commit quest resolution transaction")
		return
	}

	// Fetch updated post
	var post models.Post
	var authorXP int
	_ = h.DB.QueryRow(`
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

	u := models.User{XP: authorXP}
	u.ComputeLevelAndTitle()
	post.AuthorLevel = u.Level

	respondJSON(w, http.StatusOK, ResolveResponse{
		Post:    post,
		Rewards: rewardsSummary,
		Message: "Quest solved! Party XP has been bestowed to all contributors.",
	})
}

// GetLeaderboard returns the top adventurers by XP
func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT id, username, xp, created_at
		FROM users
		ORDER BY xp DESC, created_at ASC
		LIMIT 20
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to query leaderboard")
		return
	}
	defer rows.Close()

	leaderboard := make([]models.User, 0)
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.XP, &user.CreatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to read leaderboard entry")
			return
		}
		user.ComputeLevelAndTitle()
		leaderboard = append(leaderboard, user)
	}

	respondJSON(w, http.StatusOK, leaderboard)
}
