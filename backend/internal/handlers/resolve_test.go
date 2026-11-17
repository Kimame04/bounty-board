package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bounty-board/backend/internal/models"
)

func TestQuestResolutionAndPartyXPEngine(t *testing.T) {
	r, teardown := setupTestRouter(t)
	defer teardown()

	// Helper to login a user
	login := func(name string) models.User {
		payload := fmt.Sprintf(`{"username": "%s"}`, name)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(payload))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		var u models.User
		json.Unmarshal(rec.Body.Bytes(), &u)
		return u
	}

	alice := login("Alice")     // Quest Author
	bob := login("Bob")         // Solution Author
	charlie := login("Charlie") // Party Commenter

	// 1. Create a Topic
	topicReq := httptest.NewRequest("POST", "/api/topics", bytes.NewBufferString(`{"name": "Dungeons"}`))
	topicRec := httptest.NewRecorder()
	r.ServeHTTP(topicRec, topicReq)
	var topic models.Topic
	json.Unmarshal(topicRec.Body.Bytes(), &topic)

	// 2. Alice creates a Quest
	postPayload := fmt.Sprintf(`{"author_id": %d, "title": "Defeat Dragon", "content": "How do we slay it?"}`, alice.ID)
	postReq := httptest.NewRequest("POST", fmt.Sprintf("/api/topics/%d/posts", topic.ID), bytes.NewBufferString(postPayload))
	postRec := httptest.NewRecorder()
	r.ServeHTTP(postRec, postReq)
	var post models.Post
	json.Unmarshal(postRec.Body.Bytes(), &post)

	// 3. Bob comments with the winning solution
	bobCommentReq := httptest.NewRequest("POST", fmt.Sprintf("/api/posts/%d/comments", post.ID),
		bytes.NewBufferString(fmt.Sprintf(`{"author_id": %d, "content": "Aim for the exposed underbelly."}`, bob.ID)))
	bobCommentRec := httptest.NewRecorder()
	r.ServeHTTP(bobCommentRec, bobCommentReq)
	var bobComment models.Comment
	json.Unmarshal(bobCommentRec.Body.Bytes(), &bobComment)

	// 4. Charlie comments with extra party clues
	charlieCommentReq := httptest.NewRequest("POST", fmt.Sprintf("/api/posts/%d/comments", post.ID),
		bytes.NewBufferString(fmt.Sprintf(`{"author_id": %d, "content": "And bring fire resistance potions!"}`, charlie.ID)))
	charlieCommentRec := httptest.NewRecorder()
	r.ServeHTTP(charlieCommentRec, charlieCommentReq)

	// 5. Test Forbidden: Bob tries to resolve Alice's quest (must fail 403)
	resolveForbiddenPayload := fmt.Sprintf(`{"requester_id": %d, "accepted_comment_id": %d}`, bob.ID, bobComment.ID)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/posts/%d/resolve", post.ID), bytes.NewBufferString(resolveForbiddenPayload))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when non-author resolves, got %d", rec.Code)
	}

	// 6. Alice resolves quest, crowning Bob's answer
	resolvePayload := fmt.Sprintf(`{"requester_id": %d, "accepted_comment_id": %d}`, alice.ID, bobComment.ID)
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/posts/%d/resolve", post.ID), bytes.NewBufferString(resolvePayload))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ResolveQuest failed with status %d: %s", rec.Code, rec.Body.String())
	}

	var res ResolveResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("Failed to parse resolve response: %v", err)
	}

	if !res.Post.IsResolved {
		t.Errorf("Expected post to be marked resolved")
	}

	// Verify XP distributions
	var bobXP, charlieXP, aliceXP int
	for _, rew := range res.Rewards {
		switch rew.UserID {
		case bob.ID:
			bobXP = rew.XPAwarded
		case charlie.ID:
			charlieXP = rew.XPAwarded
		case alice.ID:
			aliceXP = rew.XPAwarded
		}
	}

	if bobXP != 50 {
		t.Errorf("Expected Bob (accepted solver) to receive 50 XP, got %d", bobXP)
	}
	if charlieXP != 15 {
		t.Errorf("Expected Charlie (party contributor) to receive 15 XP, got %d", charlieXP)
	}
	if aliceXP != 10 {
		t.Errorf("Expected Alice (quest poster) to receive 10 XP, got %d", aliceXP)
	}

	// 7. Test Idempotency: Attempting to resolve again must fail with 409 Conflict
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/posts/%d/resolve", post.ID), bytes.NewBufferString(resolvePayload))
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict when re-resolving quest, got %d", rec.Code)
	}

	// 8. Test Leaderboard
	req = httptest.NewRequest("GET", "/api/leaderboard", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GetLeaderboard failed: %s", rec.Body.String())
	}

	var leaderboard []models.User
	json.Unmarshal(rec.Body.Bytes(), &leaderboard)

	if len(leaderboard) != 3 {
		t.Fatalf("Expected 3 adventurers on leaderboard, got %d", len(leaderboard))
	}
	// Bob should be Rank 1 (50 XP)
	if leaderboard[0].Username != "Bob" || leaderboard[0].XP != 50 {
		t.Errorf("Expected Bob as #1 with 50 XP, got %+v", leaderboard[0])
	}
	// Charlie should be Rank 2 (15 XP)
	if leaderboard[1].Username != "Charlie" || leaderboard[1].XP != 15 {
		t.Errorf("Expected Charlie as #2 with 15 XP, got %+v", leaderboard[1])
	}
	// Alice should be Rank 3 (10 XP)
	if leaderboard[2].Username != "Alice" || leaderboard[2].XP != 10 {
		t.Errorf("Expected Alice as #3 with 10 XP, got %+v", leaderboard[2])
	}
}
