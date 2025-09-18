package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/2509-hackz-ichthyo/main/api/internal/config"
)

func TestMainFunc(t *testing.T) {
	t.Setenv("SERVER_PORT", "0")

	origListen := listenAndServe
	origShutdown := shutdownServer
	defer func() {
		listenAndServe = origListen
		shutdownServer = origShutdown
	}()

	listenCalled := make(chan struct{})
	listenDone := make(chan struct{})
	shutdownCalled := make(chan struct{})

	listenAndServe = func(_ *http.Server) error {
		close(listenCalled)
		<-listenDone
		return http.ErrServerClosed
	}

	shutdownServer = func(_ *http.Server, _ context.Context) error {
		close(listenDone)
		close(shutdownCalled)
		return nil
	}

	done := make(chan struct{})
	go func() {
		main()
		close(done)
	}()

	select {
	case <-listenCalled:
	case <-time.After(1 * time.Second):
		t.Fatal("listenAndServe was not invoked")
	}

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("failed to send signal: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("main() did not exit in time")
	}

	select {
	case <-shutdownCalled:
	case <-time.After(1 * time.Second):
		t.Fatal("shutdownServer was not called")
	}
}

func TestMainFuncServerError(t *testing.T) {
	t.Setenv("SERVER_PORT", "0")

	origListen := listenAndServe
	origLogFatalf := logFatalf
	defer func() {
		listenAndServe = origListen
		logFatalf = origLogFatalf
	}()

	listenAndServe = func(*http.Server) error {
		return fmt.Errorf("boom")
	}

	called := make(chan string, 1)
	logFatalf = func(format string, args ...any) {
		called <- fmt.Sprintf(format, args...)
	}

	main()

	select {
	case msg := <-called:
		if msg != "設定の読み込みに失敗しました: boom" && msg != "サーバー起動に失敗しました: boom" {
			t.Fatalf("unexpected fatal message: %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("logFatalf was not called")
	}
}

func TestRunConfigError(t *testing.T) {
	origLoad := loadConfig
	defer func() { loadConfig = origLoad }()

	loadConfig = func() (config.Config, error) {
		return config.Config{}, fmt.Errorf("load fail")
	}

	if err := run(context.Background()); err == nil || err.Error() != "設定の読み込みに失敗しました: load fail" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunShutdownError(t *testing.T) {
	origLoad := loadConfig
	origListen := listenAndServe
	origShutdown := shutdownServer
	defer func() {
		loadConfig = origLoad
		listenAndServe = origListen
		shutdownServer = origShutdown
	}()

	loadConfig = func() (config.Config, error) {
		return config.Config{ServerPort: "0"}, nil
	}

	listenCalled := make(chan struct{})
	listenDone := make(chan struct{})

	listenAndServe = func(*http.Server) error {
		close(listenCalled)
		<-listenDone
		return http.ErrServerClosed
	}

	shutdownServer = func(*http.Server, context.Context) error {
		close(listenDone)
		return fmt.Errorf("shutdown fail")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := run(ctx); err == nil || err.Error() != "サーバーの正常終了に失敗しました: shutdown fail" {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-listenCalled:
	case <-time.After(time.Second):
		t.Fatal("listenAndServe was not invoked")
	}
}

func TestRunServerErrorAfterShutdown(t *testing.T) {
	origLoad := loadConfig
	origListen := listenAndServe
	origShutdown := shutdownServer
	defer func() {
		loadConfig = origLoad
		listenAndServe = origListen
		shutdownServer = origShutdown
	}()

	loadConfig = func() (config.Config, error) {
		return config.Config{ServerPort: "0"}, nil
	}

	listenCalled := make(chan struct{})
	listenDone := make(chan struct{})

	listenAndServe = func(*http.Server) error {
		close(listenCalled)
		<-listenDone
		return fmt.Errorf("after shutdown")
	}

	shutdownServer = func(*http.Server, context.Context) error {
		close(listenDone)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := run(ctx)
	if err == nil || err.Error() != "サーバー起動に失敗しました: after shutdown" {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-listenCalled:
	case <-time.After(time.Second):
		t.Fatal("listenAndServe was not invoked")
	}
}
