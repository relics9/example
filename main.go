// Example API Server
// 正常API・バグAPI を提供し、Cloud Logging にエラーを記録するサンプル
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// ==============================================================================
// 構造化ロガー (Cloud Logging 対応)
// ==============================================================================

func logError(msg string, attrs ...any) {
	slog.Error(msg, attrs...)
}

func logInfo(msg string, attrs ...any) {
	slog.Info(msg, attrs...)
}

// ==============================================================================
// データ
// ==============================================================================

type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

var items = []Item{
	{ID: 1, Name: "Widget A", Price: 100},
	{ID: 2, Name: "Widget B", Price: 200},
	{ID: 3, Name: "Widget C", Price: 300},
}

// バグ再現用: nilマップ
var brokenConfig map[string]string

// ==============================================================================
// ハンドラー
// ==============================================================================

// GET /api/health - 正常: ヘルスチェック
func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// GET /api/items - 正常: アイテム一覧
func handleItems(w http.ResponseWriter, r *http.Request) {
	logInfo("GET /api/items", "count", len(items))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GET /api/items/{id} - 正常: ID=1〜3 / バグ: ID=0 でnilポインタパニック
func handleItemByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/items/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// BUG: id=0 のとき items[-1] でindex out of range
	item := items[id-1]

	logInfo("GET /api/items/:id", "id", id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// GET /api/divide?a=10&b=2 - 正常: a/b を返す / バグ: b=0 でゼロ除算パニック
func handleDivide(w http.ResponseWriter, r *http.Request) {
	aStr := r.URL.Query().Get("a")
	bStr := r.URL.Query().Get("b")

	a, _ := strconv.Atoi(aStr)
	b, _ := strconv.Atoi(bStr)

	// BUG: b=0 のとき division by zero パニック
	result := a / b

	logInfo("GET /api/divide", "a", a, "b", b, "result", result)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"result": result})
}

// GET /api/config - バグ: nilマップへのアクセスでパニック
func handleConfig(w http.ResponseWriter, r *http.Request) {
	// BUG: brokenConfig は nil なので panic
	value := brokenConfig["db_host"]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"db_host": value})
}

// ==============================================================================
// パニックリカバリーミドルウェア (Cloud Logging にエラーを記録)
// ==============================================================================

func withRecovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				errMsg := fmt.Sprintf("panic recovered: %v", rec)
				logError(errMsg,
					"path", r.URL.Path,
					"method", r.Method,
					"query", r.URL.RawQuery,
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

// ==============================================================================
// ルーティング
// ==============================================================================

func main() {
	// Cloud Logging 向けに JSON 形式で出力 (level→severity, msg→message にリネーム)
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				a.Key = "severity"
			}
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	})))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// 正常系
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/items", handleItems)

	// バグあり (パニックをリカバリーしてエラーログを出力)
	mux.HandleFunc("/api/items/", withRecovery(handleItemByID))
	mux.HandleFunc("/api/divide", withRecovery(handleDivide))
	mux.HandleFunc("/api/config", withRecovery(handleConfig))

	logInfo("Server starting", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logError("Server failed", "error", err)
		os.Exit(1)
	}
}
