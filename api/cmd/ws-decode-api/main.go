package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/2509-hackz-ichthyo/main/api/internal/app"
	"github.com/2509-hackz-ichthyo/main/api/internal/config"
	"github.com/2509-hackz-ichthyo/main/api/internal/domain"
	"github.com/2509-hackz-ichthyo/main/api/internal/server/httpserver"
)

// main は HTTP サーバーを起動し、Whitespace デコーダ API を提供する。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	decoder := domain.NewWhitespaceDecoder()
	usecase := app.NewDecoderUsecase(decoder)
	router := httpserver.NewRouter(usecase)

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
