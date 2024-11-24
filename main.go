package main

import (
	"fmt"
	"net/http"

	"github.com/MaximKlimenko/test_task_effective_mobile/config"
	"github.com/MaximKlimenko/test_task_effective_mobile/handlers"
	"github.com/MaximKlimenko/test_task_effective_mobile/migrations"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
)

var (
	db  *sqlx.DB
	cfg config.Config
)

func main() {
	cfg = config.LoadConfig()

	// Логгер
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	// Подключение к базе данных
	var err error
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName)

	db, err = sqlx.Open("postgres", dsn)
	if err != nil {
		logger.Fatal("Failed to connect to the database:", err)
	}
	defer db.Close()

	// Выполнение миграций
	migrations.RunMigrations(dsn, logger)

	// Настройка маршрутов
	r := chi.NewRouter()
	r.Get("/library", handlers.GetLibrary(db))
	r.Post("/add", handlers.AddSong(db, cfg))
	r.Get("/lyrics", handlers.GetSongLyrics(db))
	r.Delete("/delete/{id}", handlers.DeleteSong(db))
	r.Put("/update/{id}", handlers.UpdateSong(db))

	// Запуск сервера
	logger.Infof("Starting server on port %s", cfg.AppPort)
	http.ListenAndServe(":"+cfg.AppPort, r)
}
