// ══════════════════════════════════════════════════════
// CHOCO HUB SNAKE - Go backend
// Chỉ dùng standard library (không cần "go get" gì cả).
// Lưu data vào 1 file JSON đơn giản (data.json).
// ══════════════════════════════════════════════════════
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ── DATA MODEL ──

type User struct {
	// Tên hiển thị đúng như lúc đăng ký (giữ hoa/thường gốc)
	DisplayName string `json:"display_name"`
	Normal      int    `json:"normal"`
	Hardcore    int    `json:"hardcore"`
}

type Store struct {
	mu sync.Mutex
	// key = username chính xác (PHÂN BIỆT hoa/thường: "Alice" và "alice" là 2 user khác nhau)
	Users map[string]*User `json:"users"`
}

const dataFile = "data.json"

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{2,20}$`)

func loadStore() *Store {
	s := &Store{Users: map[string]*User{}}
	b, err := os.ReadFile(dataFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("⚠️  Không đọc được %s: %v (bắt đầu với dữ liệu trống)", dataFile, err)
		}
		return s
	}
	if err := json.Unmarshal(b, s); err != nil {
		log.Printf("⚠️  %s bị lỗi định dạng: %v (bắt đầu với dữ liệu trống)", dataFile, err)
		return &Store{Users: map[string]*User{}}
	}
	if s.Users == nil {
		s.Users = map[string]*User{}
	}
	return s
}

// save ghi file an toàn: ghi ra file tạm rồi rename, tránh mất data nếu
// server bị tắt đột ngột giữa lúc ghi.
func (s *Store) save() error {
	b, err := json.MarshalIndent(struct {
		Users map[string]*User `json:"users"`
	}{s.Users}, "", "  ")
	if err != nil {
		return err
	}
	tmp := dataFile + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, dataFile)
}

// ── HTTP RESPONSE HELPERS ──

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"status": "error", "message": msg})
}

// ── HANDLERS ──

type registerReq struct {
	Username string `json:"username"`
}

// POST /api/register
// Đăng ký username mới. PHÂN BIỆT hoa/thường: "Alice" và "alice" là 2 tài
// khoản khác nhau. Nếu trùng CHÍNH XÁC (kể cả hoa/thường) -> 409 để client
// cho người dùng thử lại tên khác.
func (s *Store) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	name := strings.TrimSpace(req.Username)
	if !usernameRe.MatchString(name) {
		writeErr(w, http.StatusBadRequest, "Username phải 2-20 ký tự, chỉ gồm chữ/số/gạch dưới")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.Users[name]; exists {
		writeErr(w, http.StatusConflict, "Username đã được sử dụng, hãy thử tên khác")
		return
	}
	s.Users[name] = &User{DisplayName: name}
	if err := s.save(); err != nil {
		log.Printf("save error: %v", err)
		writeErr(w, http.StatusInternalServerError, "save_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

type submitScoreReq struct {
	Username string `json:"username"`
	Mode     string `json:"mode"` // "normal" | "hardcore"
	Score    int    `json:"score"`
}

// POST /api/submit-score
// Lưu điểm cao nhất (best score) cho user+mode. Chỉ cập nhật nếu điểm mới cao hơn.
func (s *Store) handleSubmitScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	var req submitScoreReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if req.Mode != "normal" && req.Mode != "hardcore" {
		writeErr(w, http.StatusBadRequest, "mode phải là normal hoặc hardcore")
		return
	}
	if req.Score < 0 {
		req.Score = 0
	}
	key := strings.TrimSpace(req.Username)

	s.mu.Lock()
	defer s.mu.Unlock()

	u, exists := s.Users[key]
	if !exists {
		writeErr(w, http.StatusNotFound, "Username chưa đăng ký")
		return
	}
	if req.Mode == "normal" {
		if req.Score > u.Normal {
			u.Normal = req.Score
		}
	} else {
		if req.Score > u.Hardcore {
			u.Hardcore = req.Score
		}
	}
	if err := s.save(); err != nil {
		log.Printf("save error: %v", err)
		writeErr(w, http.StatusInternalServerError, "save_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "success",
		"best_normal":   u.Normal,
		"best_hardcore": u.Hardcore,
	})
}

type lbEntry struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
}

// GET /api/leaderboard
// Trả về 3 bảng: normal, hardcore, combined (tổng điểm 2 mode).
func (s *Store) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	const topN = 20

	s.mu.Lock()
	normal := make([]lbEntry, 0, len(s.Users))
	hardcore := make([]lbEntry, 0, len(s.Users))
	combined := make([]lbEntry, 0, len(s.Users))
	for _, u := range s.Users {
		if u.Normal > 0 {
			normal = append(normal, lbEntry{u.DisplayName, u.Normal})
		}
		if u.Hardcore > 0 {
			hardcore = append(hardcore, lbEntry{u.DisplayName, u.Hardcore})
		}
		total := u.Normal + u.Hardcore
		if total > 0 {
			combined = append(combined, lbEntry{u.DisplayName, total})
		}
	}
	s.mu.Unlock()

	sortDesc := func(list []lbEntry) []lbEntry {
		sort.Slice(list, func(i, j int) bool { return list[i].Score > list[j].Score })
		if len(list) > topN {
			list = list[:topN]
		}
		return list
	}

	writeJSON(w, http.StatusOK, map[string][]lbEntry{
		"normal":   sortDesc(normal),
		"hardcore": sortDesc(hardcore),
		"combined": sortDesc(combined),
	})
}

func main() {
	store := loadStore()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", store.handleRegister)
	mux.HandleFunc("/api/submit-score", store.handleSubmitScore)
	mux.HandleFunc("/api/leaderboard", store.handleLeaderboard)

	// Serve toàn bộ frontend tĩnh (snake.html, style.css, static/js/*, OSTS/*)
	// từ thư mục ./public. "/" trỏ thẳng vào snake.html vì file không tên index.html.
	fs := http.FileServer(http.Dir("./public"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "./public/snake.html")
			return
		}
		fs.ServeHTTP(w, r)
	})

	addr := ":8080"
	if v := os.Getenv("PORT"); v != "" {
		addr = ":" + v
	}
	log.Printf("🐍 Snake server đang chạy tại http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
