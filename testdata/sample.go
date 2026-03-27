package sample

import (
	"fmt"
	"strings"
)

// UserService handles user operations.
type UserService struct {
	db    *Database
	cache *Cache
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(id string) (*User, error) {
	if cached, ok := s.cache.Get(id); ok {
		return cached, nil
	}
	return s.db.Find(id)
}

// DeleteUser removes a user by ID.
func (s *UserService) DeleteUser(id string) error {
	s.cache.Invalidate(id)
	return s.db.Delete(id)
}

// Config holds application configuration.
type Config struct {
	Host    string
	Port    int
	Debug   bool
	Version string
}

// Validate checks the config for errors.
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	return nil
}

func formatName(first, last string) string {
	return strings.TrimSpace(first + " " + last)
}
