package models

import "time"

// User represents an adventurer in the guild
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	XP        int       `json:"xp"`
	Level     int       `json:"level"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// ComputeLevelAndTitle calculates the RPG progression based on accumulated XP
func (u *User) ComputeLevelAndTitle() {
	// Formula: Level = floor(sqrt(XP / 10)) + 1
	// e.g. 0 XP -> Lvl 1; 40 XP -> Lvl 3; 100 XP -> Lvl 4; 900 XP -> Lvl 10
	level := 1
	for (level)*(level)*10 <= u.XP {
		level++
	}
	u.Level = level

	switch {
	case u.Level >= 15:
		u.Title = "Grand Archmage"
	case u.Level >= 10:
		u.Title = "Guildmaster"
	case u.Level >= 7:
		u.Title = "Arcane Architect"
	case u.Level >= 4:
		u.Title = "Journeyman Caster"
	default:
		u.Title = "Wandering Novice"
	}
}

// Topic represents a guild hall or discussion category
type Topic struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PostCount   int       `json:"post_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Post represents a quest or issue posted to a topic
type Post struct {
	ID                int64     `json:"id"`
	TopicID           int64     `json:"topic_id"`
	AuthorID          int64     `json:"author_id"`
	AuthorName        string    `json:"author_name,omitempty"`
	AuthorLevel       int       `json:"author_level,omitempty"`
	Title             string    `json:"title"`
	Content           string    `json:"content"`
	IsResolved        bool      `json:"is_resolved"`
	AcceptedCommentID *int64    `json:"accepted_comment_id,omitempty"`
	CommentCount      int       `json:"comment_count,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Comment represents a solution or reply to a quest
type Comment struct {
	ID          int64     `json:"id"`
	PostID      int64     `json:"post_id"`
	AuthorID    int64     `json:"author_id"`
	AuthorName  string    `json:"author_name,omitempty"`
	AuthorLevel int       `json:"author_level,omitempty"`
	Content     string    `json:"content"`
	Upvotes     int       `json:"upvotes"`
	IsAccepted  bool      `json:"is_accepted"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// QuestReward records an XP distribution event
type QuestReward struct {
	ID         int64     `json:"id"`
	PostID     int64     `json:"post_id"`
	UserID     int64     `json:"user_id"`
	XPAwarded  int       `json:"xp_awarded"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

// ResolveQuestRequest is the payload sent when a post author marks a quest helpful/resolved
type ResolveQuestRequest struct {
	AcceptedCommentID *int64 `json:"accepted_comment_id"`
}
