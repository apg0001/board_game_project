package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"board-game-platform/apps/api/internal/auth"
	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/chat"
	"board-game-platform/apps/api/internal/config"
	"board-game-platform/apps/api/internal/connection"
	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/games/davinci"
	"board-game-platform/apps/api/internal/games/halligalli"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/httpapi"
	"board-game-platform/apps/api/internal/match"
	"board-game-platform/apps/api/internal/realtime"
	"board-game-platform/apps/api/internal/record"
	"board-game-platform/apps/api/internal/room"
	"board-game-platform/apps/api/internal/session"
	"board-game-platform/apps/api/internal/tutorial"
)

func main() {
	appCtx, stopApp := context.WithCancel(context.Background())
	defer stopApp()

	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	gameCatalog := catalog.NewInMemoryCatalog(catalog.DefaultGames())
	authService := auth.NewService(time.Now)
	gameRegistry := gamecore.NewRegistry(davinci.NewModule(), halligalli.NewModule())
	guestService := guest.NewService(guest.NewMemoryStore(), time.Now)
	roomService := room.NewService(room.NewMemoryStore(), time.Now)
	sessionService := session.NewService(session.NewMemoryStore(), gameRegistry, time.Now)
	chatService := chat.NewService(time.Now, 50)
	presenceService := connection.NewService(time.Now, 60*time.Second)
	recordService := record.NewService(time.Now)
	matchService := match.NewService()
	tutorialService := tutorial.NewService()
	hub := realtime.NewHub(logger)
	go hub.Run()
	sweepExpiredPresence := httpapi.NewPresenceSweeper(gameCatalog, authService, guestService, roomService, sessionService, chatService, presenceService, recordService, matchService, tutorialService, hub)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sweepExpiredPresence(context.Background())
			case <-appCtx.Done():
				return
			}
		}
	}()

	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      httpapi.NewRouter(cfg, logger, gameCatalog, authService, guestService, roomService, sessionService, chatService, presenceService, recordService, matchService, tutorialService, hub),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("api server started", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	stopApp()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
