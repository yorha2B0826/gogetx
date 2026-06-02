package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/module"
	"gopkg.in/yaml.v3"
)

type Manager struct {
	path string
}

type File struct {
	Favorites map[string]string `yaml:"favorites"`
}

func NewManager(path string) *Manager {
	return &Manager{path: path}
}

func (m *Manager) Path() string {
	return m.path
}

func (m *Manager) Favorite(alias string) (string, bool, error) {
	file, err := m.load()
	if err != nil {
		return "", false, err
	}
	modulePath, ok := file.Favorites[alias]
	return modulePath, ok, nil
}

func (m *Manager) Favorites() (map[string]string, error) {
	file, err := m.load()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(file.Favorites))
	for key, value := range file.Favorites {
		out[key] = value
	}
	return out, nil
}

func (m *Manager) FavoriteAliases() ([]string, error) {
	favorites, err := m.Favorites()
	if err != nil {
		return nil, err
	}
	aliases := make([]string, 0, len(favorites))
	for alias := range favorites {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases, nil
}

func (m *Manager) AddFavorite(alias string, modulePath string) error {
	alias = strings.TrimSpace(alias)
	modulePath = strings.TrimSpace(modulePath)
	if alias == "" {
		return fmt.Errorf("favorite alias is required")
	}
	if err := module.CheckPath(modulePath); err != nil {
		return fmt.Errorf("invalid module path %q: %w", modulePath, err)
	}

	file, err := m.load()
	if err != nil {
		return err
	}
	file.Favorites[alias] = modulePath
	return m.save(file)
}

func (m *Manager) RemoveFavorite(alias string) error {
	file, err := m.load()
	if err != nil {
		return err
	}
	delete(file.Favorites, alias)
	return m.save(file)
}

func (m *Manager) load() (File, error) {
	file := File{Favorites: map[string]string{}}
	content, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		return file, nil
	}
	if err != nil {
		return file, err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return file, nil
	}
	if err := yaml.Unmarshal(content, &file); err != nil {
		return file, fmt.Errorf("read config: %w", err)
	}
	if file.Favorites == nil {
		file.Favorites = map[string]string{}
	}
	return file, nil
}

func (m *Manager) save(file File) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	content, err := yaml.Marshal(file)
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, content, 0o644)
}
