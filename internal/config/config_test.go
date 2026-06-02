package config

import "testing"

func TestManagerAddsListsAndRemovesFavorites(t *testing.T) {
	t.Parallel()

	manager := NewManager(t.TempDir() + "/config.yaml")
	if err := manager.AddFavorite("logger", "go.uber.org/zap"); err != nil {
		t.Fatalf("AddFavorite returned error: %v", err)
	}

	modulePath, ok, err := manager.Favorite("logger")
	if err != nil {
		t.Fatalf("Favorite returned error: %v", err)
	}
	if !ok || modulePath != "go.uber.org/zap" {
		t.Fatalf("Favorite = (%q, %v), want (go.uber.org/zap, true)", modulePath, ok)
	}

	favorites, err := manager.Favorites()
	if err != nil {
		t.Fatalf("Favorites returned error: %v", err)
	}
	if favorites["logger"] != "go.uber.org/zap" {
		t.Fatalf("favorites = %#v, missing logger", favorites)
	}

	reloaded := NewManager(manager.Path())
	modulePath, ok, err = reloaded.Favorite("logger")
	if err != nil {
		t.Fatalf("Favorite after reload returned error: %v", err)
	}
	if !ok || modulePath != "go.uber.org/zap" {
		t.Fatalf("Favorite after reload = (%q, %v), want (go.uber.org/zap, true)", modulePath, ok)
	}

	if err := reloaded.RemoveFavorite("logger"); err != nil {
		t.Fatalf("RemoveFavorite returned error: %v", err)
	}
	_, ok, err = reloaded.Favorite("logger")
	if err != nil {
		t.Fatalf("Favorite after remove returned error: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want false after removal")
	}
}
