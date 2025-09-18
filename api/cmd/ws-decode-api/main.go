package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/2509-hackz-ichthyo/main/api/internal/app"
	"github.com/2509-hackz-ichthyo/main/api/internal/config"
	"github.com/2509-hackz-ichthyo/main/api/internal/server/httpserver"
)

var (
	listenAndServe = func(srv *http.Server) error {
		return srv.ListenAndServe()
	}

	shutdownServer = func(srv *http.Server, ctx context.Context) error {
		return srv.Shutdown(ctx)
	}

	loadConfig = config.Load

	newWhitespaceUsecase = app.NewWhitespaceUsecase

	newRouter = httpserver.NewRouter

	logFatalf = log.Fatalf
)

// main は HTTP サーバーを起動し、Whitespace デコーダ API を提供する。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		logFatalf("%v", err)
	}
}

func run(ctx context.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("設定の読み込みに失敗しました: %w", err)
	}

	usecase := newWhitespaceUsecase()
	router := newRouter(usecase)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("サーバーを起動しました: http://0.0.0.0:%s", cfg.ServerPort)
		if err := listenAndServe(srv); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("サーバー起動に失敗しました: %w", err)
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
	}

	log.Println("シャットダウンシグナルを受信しました。終了処理を開始します。")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := shutdownServer(srv, shutdownCtx); err != nil {
		return fmt.Errorf("サーバーの正常終了に失敗しました: %w", err)
	}

	if err := <-serverErr; err != nil {
		return err
	}

	log.Println("サーバーを終了しました。")
	return nil
}
