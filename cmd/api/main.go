package main

import (
	"context"
	"log"
	"net/http"
	"os"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := db.RunMigrations(cfg.DatabaseURL, "/app/migrations"); err != nil {
		log.Fatal("migrations: ", err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db: ", err)
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
		Addr:    ":" + cfg.Port,
		Handler: router,
	}
	
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Fatal("shutdown: ", err)
		}
	}()

	log.Printf("listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
