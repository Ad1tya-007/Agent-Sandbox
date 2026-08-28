package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/api"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/kubernetes"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/logs"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/resources"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/sandbox"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/watcher"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/websocket"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := envOr("AGENT_SANDBOX_ADDR", "127.0.0.1:8787")
	k8s := kubernetes.New(os.Getenv("KUBECONFIG"), envOr("AGENT_SANDBOX_NAMESPACE", ""))

	sandboxes := sandbox.New(k8s)
	mapper := resources.New(k8s)
	watch := watcher.New(k8s, mapper)
	logMgr := logs.New(k8s, mapper)
	hub := websocket.New(watch)
	watch.Start(ctx)

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(sandboxes, hub, logMgr).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := k8s.Err(); err != nil {
			log.Printf("kubernetes: %v (HTTP still serving /healthz)", err)
		} else {
			log.Printf("kubernetes: context %q namespace %q", k8s.ClusterName(), k8s.Namespace())
		}
		log.Printf("listening on http://%s", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := srv.Shutdown(shutdownCtx)
		hub.Close()
		watch.Stop()
		logMgr.Close()
		return err
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
