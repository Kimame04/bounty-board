package database

import (
	"os"
	"testing"
)

func TestInitDBAndSchema(t *testing.T) {
	testDBPath := "test_bounty.db"
	defer os.Remove(testDBPath)

	db, err := InitDB(testDBPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// 1. Verify tables exist
	tables := []string{"users", "topics", "posts", "comments", "quest_rewards"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil || name != table {
			t.Errorf("Expected table %s to exist, but got error: %v", table, err)
		}
	}

	// 2. Test inserting user
	res, err := db.Exec("INSERT INTO users (username, xp) VALUES (?, ?)", "Galahad", 100)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}
	userID, _ := res.LastInsertId()

	// 3. Test inserting topic
	res, err = db.Exec("INSERT INTO topics (name, description) VALUES (?, ?)", "Alchemy", "Potion bugs")
	if err != nil {
		t.Fatalf("Failed to insert topic: %v", err)
	}
	topicID, _ := res.LastInsertId()

	// 4. Test inserting post
	res, err = db.Exec("INSERT INTO posts (topic_id, author_id, title, content) VALUES (?, ?, ?, ?)",
		topicID, userID, "Exploding Cauldron", "Why does it blow up?")
	if err != nil {
		t.Fatalf("Failed to insert post: %v", err)
	}
	postID, _ := res.LastInsertId()

	// 5. Test inserting comment
	res, err = db.Exec("INSERT INTO comments (post_id, author_id, content) VALUES (?, ?, ?)",
		postID, userID, "Use less mandrake root.")
	if err != nil {
		t.Fatalf("Failed to insert comment: %v", err)
	}

	// 6. Test quest reward idempotency (UNIQUE constraint)
	_, err = db.Exec("INSERT INTO quest_rewards (post_id, user_id, xp_awarded, reason) VALUES (?, ?, ?, ?)",
		postID, userID, 50, "accepted_solution")
	if err != nil {
		t.Fatalf("Failed to insert first reward: %v", err)
	}

	// Duplicate insert with same (post_id, user_id, reason) MUST fail
	_, err = db.Exec("INSERT INTO quest_rewards (post_id, user_id, xp_awarded, reason) VALUES (?, ?, ?, ?)",
		postID, userID, 50, "accepted_solution")
	if err == nil {
		t.Errorf("Expected duplicate quest reward to fail UNIQUE constraint, but it succeeded")
	}

	// 7. Test foreign key cascade delete
	_, err = db.Exec("DELETE FROM topics WHERE id = ?", topicID)
	if err != nil {
		t.Fatalf("Failed to delete topic: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ?", postID).Scan(&count)
	if count != 0 {
		t.Errorf("Expected cascade deletion of posts when topic deleted, found %d remaining", count)
	}
}
