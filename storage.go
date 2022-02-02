package jsonstorage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/thamaji/fstools"
)

func New[T any](dir string) *Storage[T] {
	return &Storage[T]{dir: dir, mutex: sync.RWMutex{}}
}

type Storage[T any] struct {
	dir   string
	mutex sync.RWMutex
}

type entry[T any] struct {
	Key   string `json:"key"`
	Value T      `json:"value"`
}

func (storage *Storage[T]) Range(f func(string, T) error) error {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	direntries, err := fstools.ReadDir(storage.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("%w: failed to range JSONs: %s", ErrInternal, err)
	}

	for _, direntry := range direntries {
		if direntry.IsDir() {
			continue
		}

		name := direntry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}

		path := filepath.Join(storage.dir, name)
		entry, err := storage.get(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("%w: failed to range JSONs: %s", ErrInternal, err)
		}

		if err := f(entry.Key, entry.Value); err != nil {
			return err
		}
	}

	return nil
}

func (storage *Storage[T]) Get(key string) (T, error) {
	key = strings.ToLower(key)
	path := filepath.Join(storage.dir, url.PathEscape(key)+".json")
	storage.mutex.RLock()
	entry, err := storage.get(path)
	storage.mutex.RUnlock()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return *new(T), fmt.Errorf("%w: %s", ErrNotExist, key)
		}
		return *new(T), fmt.Errorf("%w: failed to get JSON: %s", ErrInternal, err)
	}
	return entry.Value, nil
}

func (storage *Storage[T]) get(path string) (*entry[T], error) {
	entry := entry[T]{}
	err := fstools.ReadFileFunc(path, func(r io.Reader) error {
		return json.NewDecoder(r).Decode(&entry)
	})
	return &entry, err
}

func (storage *Storage[T]) Put(key string, value T) error {
	key = strings.ToLower(key)
	path := filepath.Join(storage.dir, url.PathEscape(key)+".json")
	storage.mutex.Lock()
	err := fstools.WriteFileFunc(path, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(entry[T]{
			Key:   key,
			Value: value,
		})
	})
	storage.mutex.Unlock()
	if err != nil {
		return fmt.Errorf("%w: failed to put JSON: %s", ErrInternal, err)
	}
	return nil
}

func (storage *Storage[T]) Delete(key string) error {
	key = strings.ToLower(key)
	path := filepath.Join(storage.dir, url.PathEscape(key)+".json")
	var err error
	storage.mutex.Lock()
	if fstools.Exists(path) {
		err = os.Remove(path)
	}
	storage.mutex.Unlock()
	if err != nil {
		return fmt.Errorf("%w: failed to delete JSON: %s", ErrInternal, err)
	}
	return nil
}
