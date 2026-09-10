package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"wishlist-api/internal/config"
	"wishlist-api/internal/db"
	"wishlist-api/internal/handler"
	myjwt "wishlist-api/internal/jwt"
	"wishlist-api/internal/repository"
	"wishlist-api/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := db.RunMigrations(cfg.DatabaseURL, cfg.MigrationsDir); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	defer pool.Close()

	jwtManager := myjwt.NewManager(cfg.JWTSecret, cfg.JWTExpiry)

	userRepo := repository.NewUserRepo(pool.PgxPool())
	wishlistRepo := repository.NewWishlistRepo(pool.PgxPool())
	itemRepo := repository.NewItemRepo(pool.PgxPool())

	authService := service.NewAuthService(userRepo, jwtManager)
	wishlistService := service.NewWishlistService(wishlistRepo, itemRepo)
	itemService := service.NewItemService(itemRepo, wishlistRepo)
	publicService := service.NewPublicService(wishlistRepo, itemRepo)

	authHandler := handler.NewAuthHandler(authService)
	wishlistHandler := handler.NewWishlistHandler(wishlistService)
	itemHandler := handler.NewItemHandler(itemService)
	publicHandler := handler.NewPublicHandler(publicService)

	router := handler.NewRouter(jwtManager, authHandler, wishlistHandler, itemHandler, publicHandler)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	log.Printf("listening on :%s", cfg.Port)
	return serve(ctx, srv, listener, 10*time.Second)
}

func serve(ctx context.Context, srv *http.Server, listener net.Listener, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			err = errors.Join(err, srv.Close())
		}
		serveErr := <-done
		if !errors.Is(serveErr, http.ErrServerClosed) {
			err = errors.Join(err, serveErr)
		}
		return err
	}
}
