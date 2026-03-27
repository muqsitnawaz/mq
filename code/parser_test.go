package code

import (
	"testing"

	mq "github.com/muqsitnawaz/mq/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var goSample = []byte(`package main

import "fmt"

type UserService struct {
	db    *Database
	cache *Cache
}

func (s *UserService) GetUser(id string) (*User, error) {
	return s.db.Find(id)
}

func (s *UserService) DeleteUser(id string) error {
	return s.db.Delete(id)
}

type Config struct {
	Host string
	Port int
}

func main() {
	fmt.Println("hello")
}
`)

var pySample = []byte(`import os
from dataclasses import dataclass
from typing import Optional

@dataclass
class Config:
    host: str
    port: int = 8080
    debug: bool = False

class UserRepository:
    def __init__(self, db):
        self.db = db

    def find_by_id(self, user_id: str) -> Optional[dict]:
        return self.db.query("SELECT * FROM users WHERE id = ?", user_id)

def create_app(config: Config):
    app = Flask(__name__)
    return app
`)

var tsSample = []byte(`import { useState } from 'react';

interface UserProps {
  name: string;
  age: number;
}

export function UserCard({ name, age }: UserProps) {
  const [expanded, setExpanded] = useState(false);
  return expanded;
}

export class UserService {
  private api: ApiClient;

  async getUser(id: string): Promise<User> {
    return this.api.get('/users/' + id);
  }
}

export const MAX_RETRIES = 3;
`)

var rustSample = []byte(`use std::collections::HashMap;

pub struct Registry {
    items: HashMap<String, Item>,
}

pub trait Processor {
    fn process(&self, input: &[u8]) -> Result<Vec<u8>, Error>;
    fn name(&self) -> &str;
}

impl Registry {
    pub fn new() -> Self {
        Self { items: HashMap::new() }
    }
}

pub fn init_logging(level: &str) {
    env_logger::Builder::new().init();
}

pub enum Status {
    Active,
    Inactive,
}
`)

func TestGoParser(t *testing.T) {
	p := NewParser()
	doc, err := p.Parse(goSample, "main.go")
	require.NoError(t, err)

	assert.Equal(t, mq.FormatCode, doc.Format())
	assert.Equal(t, "go: main.go", doc.Title())

	headings := doc.GetHeadings()
	require.True(t, len(headings) >= 5, "expected at least 5 headings, got %d", len(headings))

	// Check that we get type declarations as H1
	h1s := doc.GetHeadings(1)
	typeNames := make([]string, len(h1s))
	for i, h := range h1s {
		typeNames[i] = h.Text
	}
	assert.Contains(t, typeNames[0], "type UserService struct")
	assert.Contains(t, typeNames[1], "type Config struct")

	// Check that we get functions/methods as H2
	h2s := doc.GetHeadings(2)
	assert.True(t, len(h2s) >= 3, "expected at least 3 H2 headings for methods+functions")

	// Check sections have correct line ranges
	sections := doc.GetSections()
	assert.True(t, len(sections) > 0)
	for _, s := range sections {
		assert.True(t, s.Start > 0, "section should have start line")
		assert.True(t, s.End >= s.Start, "section end should be >= start")
	}
}

func TestPythonParser(t *testing.T) {
	p := NewParser()
	doc, err := p.Parse(pySample, "service.py")
	require.NoError(t, err)

	assert.Equal(t, "python: service.py", doc.Title())

	// @dataclass wraps Config -- decorator should unwrap to find class
	h1s := doc.GetHeadings(1)
	require.True(t, len(h1s) >= 2, "expected at least 2 classes")

	h2s := doc.GetHeadings(2)
	assert.True(t, len(h2s) >= 1, "expected at least 1 standalone function")
}

func TestTypeScriptParser(t *testing.T) {
	p := NewParser()
	doc, err := p.Parse(tsSample, "app.tsx")
	require.NoError(t, err)

	assert.Contains(t, doc.Title(), "app.tsx")

	headings := doc.GetHeadings()
	assert.True(t, len(headings) >= 4, "expected interface, function, class, const")

	// Should have interface and class as H1
	h1s := doc.GetHeadings(1)
	assert.True(t, len(h1s) >= 1, "expected at least 1 H1 (interface or class)")
}

func TestRustParser(t *testing.T) {
	p := NewParser()
	doc, err := p.Parse(rustSample, "lib.rs")
	require.NoError(t, err)

	assert.Equal(t, "rust: lib.rs", doc.Title())

	h1s := doc.GetHeadings(1)
	require.True(t, len(h1s) >= 3, "expected struct, trait, impl, enum as H1")

	h2s := doc.GetHeadings(2)
	assert.True(t, len(h2s) >= 1, "expected at least 1 function as H2")
}

func TestDetectCode(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"main.go", true},
		{"app.tsx", true},
		{"service.py", true},
		{"lib.rs", true},
		{"Main.java", true},
		{"style.css", true},
		{"README.md", false}, // markdown, not code
		{"data.json", false}, // json, not code
		{"unknown.xyz", false},
	}

	for _, tt := range tests {
		f, ok := detectCode(tt.path, nil)
		if tt.want {
			assert.True(t, ok, "%s should be detected as code", tt.path)
			assert.Equal(t, mq.FormatCode, f)
		} else {
			assert.False(t, ok, "%s should NOT be detected as code", tt.path)
		}
	}
}

func TestCodeBlocks(t *testing.T) {
	p := NewParser()
	doc, err := p.Parse(goSample, "main.go")
	require.NoError(t, err)

	blocks := doc.GetCodeBlocks()
	require.Len(t, blocks, 1)
	assert.Equal(t, "go", blocks[0].Language)
	assert.Contains(t, blocks[0].Content, "package main")
}
