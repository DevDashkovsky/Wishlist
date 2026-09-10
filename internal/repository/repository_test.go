package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"wishlist-api/internal/domain"
)

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("WISHLIST_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("WISHLIST_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	schema := "wishlist_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	paths, err := filepath.Glob("../../migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		up := strings.Split(string(data), "-- +goose Down")[0]
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatal(err)
		}
	}
	return pool
}

func TestRepositoryMutations(t *testing.T) {
	pool := integrationPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	users, lists, items := NewUserRepo(pool), NewWishlistRepo(pool), NewItemRepo(pool)
	user := &domain.User{Email: "test@example.com", PasswordHash: "hash"}
	if err := users.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	list := &domain.Wishlist{UserID: user.ID, Title: "Original", Description: "old", EventDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ShareToken: uuid.NewString()}
	if err := lists.Create(ctx, list); err != nil {
		t.Fatal(err)
	}
	item := &domain.Item{WishlistID: list.ID, Title: "Original", Priority: 3}
	if err := items.Create(ctx, item); err != nil {
		t.Fatal(err)
	}
	title, description := "new title", "new description"
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for _, fn := range []func() error{
		func() error { _, err := lists.Patch(ctx, list.ID, domain.WishlistChanges{Title: &title}); return err },
		func() error {
			_, err := lists.Patch(ctx, list.ID, domain.WishlistChanges{Description: &description})
			return err
		},
		func() error { _, err := items.Patch(ctx, item.ID, domain.ItemPatch{Title: &title}); return err },
		func() error {
			_, err := items.Patch(ctx, item.ID, domain.ItemPatch{Description: &description})
			return err
		},
	} {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- fn() }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	gotList, err := lists.GetByID(ctx, list.ID)
	if err != nil {
		t.Fatal(err)
	}
	gotItem, err := items.GetByID(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotList.Title != title || gotList.Description != description || gotItem.Title != title || gotItem.Description != description {
		t.Fatal("disjoint patches lost changes")
	}

	reserveErrors := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := items.Reserve(ctx, item.ID); reserveErrors <- err }()
	}
	wg.Wait()
	close(reserveErrors)
	success, conflict := 0, 0
	for err := range reserveErrors {
		switch {
		case err == nil:
			success++
		case errors.Is(err, domain.ErrAlreadyReserved):
			conflict++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
	if err := items.Update(ctx, item); err != nil {
		t.Fatal(err)
	}
	if !item.IsReserved {
		t.Fatal("update returned stale reservation")
	}
	patched, err := items.Patch(ctx, item.ID, domain.ItemPatch{Title: &title})
	if err != nil || !patched.IsReserved {
		t.Fatalf("patch=%+v err=%v", patched, err)
	}

	if err := items.Delete(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	for _, fn := range []func() error{
		func() error { return items.Update(ctx, item) },
		func() error { _, err := items.Patch(ctx, item.ID, domain.ItemPatch{Title: &title}); return err },
		func() error { return items.Delete(ctx, item.ID) },
		func() error { _, err := items.Reserve(ctx, item.ID); return err },
	} {
		if err := fn(); !errors.Is(err, domain.ErrItemNotFound) {
			t.Fatalf("missing item: %v", err)
		}
	}
	if err := lists.Delete(ctx, list.ID); err != nil {
		t.Fatal(err)
	}
	for _, fn := range []func() error{
		func() error { return lists.Update(ctx, list) },
		func() error { _, err := lists.Patch(ctx, list.ID, domain.WishlistChanges{Title: &title}); return err },
		func() error { return lists.Delete(ctx, list.ID) },
		func() error {
			return items.Create(ctx, &domain.Item{WishlistID: list.ID, Title: "orphan", Priority: 3})
		},
	} {
		if err := fn(); !errors.Is(err, domain.ErrWishlistNotFound) {
			t.Fatalf("missing wishlist: %v", err)
		}
	}
}
