// Command server runs the SSH portfolio.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"charm.land/ssh"

	"github.com/namelessmonarch0/ssh-portfolio/content"
	"github.com/namelessmonarch0/ssh-portfolio/internal/server"
	"github.com/namelessmonarch0/ssh-portfolio/internal/vfs"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := server.ConfigFromEnv(os.Getenv)
	if err != nil {
		return err
	}
	fsys, err := vfs.New(content.Files)
	if err != nil {
		return err
	}
	profile, err := content.LoadProfile()
	if err != nil {
		return err
	}
	srv, err := server.New(cfg, fsys, profile, log)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.ListenAddr)
		errc <- srv.SSH.ListenAndServe()
	}()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if err := srv.SSH.Close(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		return err
	}
	return nil
}
