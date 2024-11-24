package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/MaximKlimenko/test_task_effective_mobile/config"
	"github.com/MaximKlimenko/test_task_effective_mobile/models"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// getLibrary — получение данных библиотеки с фильтрацией и пагинацией
func GetLibrary(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := "SELECT * FROM songs WHERE 1=1"
		params := []interface{}{}
		i := 1

		// Фильтры
		if group := r.URL.Query().Get("group"); group != "" {
			query += fmt.Sprintf(" AND group_name = $%d", i)
			params = append(params, group)
			i++
		}
		if song := r.URL.Query().Get("song"); song != "" {
			query += fmt.Sprintf(" AND song_name = $%d", i)
			params = append(params, song)
			i++
		}

		// Пагинация
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if limit == 0 {
			limit = 10 // значение по умолчанию
		}
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", i, i+1)
		params = append(params, limit, offset)

		// Выполнение запроса
		var songs []models.Song
		err := db.Select(&songs, query, params...)
		if err != nil {
			http.Error(w, "Failed to fetch library", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(songs)
	}
}

// getSongLyrics — получение текста песни с пагинацией по куплетам
func GetSongLyrics(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			http.Error(w, "Invalid song ID", http.StatusBadRequest)
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		if limit == 0 {
			limit = 1 // значение по умолчанию
		}

		// Получение песни
		var song models.Song
		err = db.Get(&song, "SELECT * FROM songs WHERE id = $1", id)
		if err != nil {
			http.Error(w, "Song not found", http.StatusNotFound)
			return
		}

		// Пагинация по куплетам
		start := offset
		end := start + limit
		if start >= len(song.Lyrics) {
			http.Error(w, "No more verses", http.StatusNotFound)
			return
		}
		if end > len(song.Lyrics) {
			end = len(song.Lyrics)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(song.Lyrics[start:end])
	}
}

// addSong — добавление новой песни
func AddSong(db *sqlx.DB, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.AddSongRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		// Запрос в внешнее API
		releaseDate, lyrics, link, err := fetchSongDetails(req.Group, req.Song, cfg)
		if err != nil {
			http.Error(w, "Failed to fetch song details", http.StatusInternalServerError)
			return
		}

		// Сохранение в БД
		_, err = db.Exec(`
		INSERT INTO songs (group_name, song_name, release_date, lyrics, link)
		VALUES ($1, $2, $3, $4, $5)`,
			req.Group, req.Song, releaseDate, lyrics, link)
		if err != nil {
			http.Error(w, "Failed to save song", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Song added successfully"})
	}
}

// deleteSong — удаление песни
func DeleteSong(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		_, err := db.Exec("DELETE FROM songs WHERE id = $1", id)
		if err != nil {
			http.Error(w, "Failed to delete song", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Song deleted successfully"})
	}
}

// updateSong — изменение данных песни
func UpdateSong(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var song models.Song
		if err := json.NewDecoder(r.Body).Decode(&song); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		_, err := db.Exec(`
		UPDATE songs
		SET group_name = $1, song_name = $2, release_date = $3, lyrics = $4, link = $5
		WHERE id = $6`,
			song.Group, song.Song, song.ReleaseDate, song.Lyrics, song.Link, id)
		if err != nil {
			http.Error(w, "Failed to update song", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Song updated successfully"})
	}
}

// fetchSongDetails — получение обогащённой информации о песне
func fetchSongDetails(group, song string, cfg config.Config) (releaseDate string, lyrics []string, link string, err error) {
	apiURL := fmt.Sprintf("%s?group=%s&song=%s", cfg.APIUrl, group, song)
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, "", fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var details struct {
		ReleaseDate string `json:"releaseDate"`
		Text        string `json:"text"`
		Link        string `json:"link"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return "", nil, "", err
	}

	lyrics = splitLyrics(details.Text)
	return details.ReleaseDate, lyrics, details.Link, nil
}

// splitLyrics — разбивка текста песни на куплеты
func splitLyrics(text string) []string {
	return strings.Split(text, "\n\n")
}
