package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/2509-hackz-ichthyo/main/api/internal/config"
	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
	"github.com/2509-hackz-ichthyo/main/api/internal/infrastructure/database"
	"github.com/2509-hackz-ichthyo/main/api/internal/infrastructure/repository"
	"github.com/2509-hackz-ichthyo/main/api/internal/interfaces/httpapi"
	"github.com/2509-hackz-ichthyo/main/api/internal/usecases"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	db, err := database.OpenSQLite(database.SQLiteConfig{
		Path:            cfg.DatabasePath,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 0,
	})
	if err != nil {
		log.Fatalf("データベース接続に失敗しました: %v", err)
	}
	defer db.Close()

	if err := database.EnsureSchema(ctx, db); err != nil {
		log.Fatalf("スキーマの初期化に失敗しました: %v", err)
	}

	repo := repository.NewSQLiteCommandRepository(db)
	decoder := domain.NewWhitespaceDecoder()
	encoder := domain.NewWhitespaceEncoder()
	usecase := usecases.NewCommandExecutor(repo, decoder, encoder)
	router := httpapi.NewRouter(usecase)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	go func() {
		log.Printf("サーバーを起動しました: http://0.0.0.0:%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("サーバー起動に失敗しました: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("シャットダウンシグナルを受信しました。終了処理を開始します。")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("サーバーの正常終了に失敗しました: %v", err)
	}

	log.Println("サーバーを終了しました。")
}
