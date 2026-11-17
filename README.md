# BountyBoard 📜⚔️

A fantasy-themed Q&A forum built from scratch to teach myself the fundamentals of full-stack web development: Go backends, relational databases, RESTful architecture, and React with TypeScript.

Discussions are structured like adventurer guild quests: users post bounties (questions) and submit dispatches (solutions). When a quest is solved, the party shares the XP rewards!

---

## 🎯 Learning Goals
* **Backend Fundamentals:** Writing idiomatic Go with `net/http` and `chi`.
* **Database & SQL:** Designing relational schemas, indexes, foreign key cascades, and ACID transactions in SQLite.
* **API Design:** Clean RESTful conventions with JSON payloads and input validation.
* **Frontend Engineering:** React 18, TypeScript, Material UI, and custom HTML5 canvas rendering for procedural 8-bit avatars.

---

## 🛠️ Stack
* **Backend:** Go 1.22+ (`go-chi/chi/v5`, `modernc.org/sqlite`)
* **Database:** SQLite (WAL mode, foreign keys enabled)
* **Frontend:** React 18 + TypeScript (Vite)
* **Avatars:** Procedural 8-bit sprite generator driven by username hashes & XP levels

---

## 🚀 Quickstart

### Option A: Standalone Single-Binary (Recommended)
You can compile the entire full-stack app into a **single 11 MB standalone binary**:
```bash
./build.sh
./bounty-board-app
# Open http://localhost:8080 in your browser!
```

### Option B: Development Mode (Hot Reloading)
1. **Backend:**
   ```bash
   cd backend
   go run cmd/server/main.go
   ```
2. **Frontend:**
   ```bash
   cd frontend
   npm run dev
   # Open http://localhost:5173
   ```

### Run Tests
```bash
cd backend
go test -v ./...
```
