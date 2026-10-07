// Package domain holds the core models shared by every layer of GitShiny.
package domain

import "time"

// FileChange is one file touched by a commit (from `git log --numstat`).
type FileChange struct {
	Path    string
	Added   int
	Removed int
	Binary  bool
}

// Commit is a single commit with the files it changed.
type Commit struct {
	Hash   string
	Author string
	Email  string
	Time   time.Time
	Files  []FileChange
}

// Query describes which commits to analyse.
type Query struct {
	Authors     []string // empty = all authors
	Since       time.Time
	Until       time.Time
	Period      string // human label: Today, Yesterday, Custom
	Branch      string // empty = current branch (HEAD)
	AllBranches bool
	Paths       []string // git pathspecs to include; empty = everything
	Exclude     []string // git pathspecs to leave out
	GroupBy     string   // "", "author", "day" or "file"
}

// Group is one row of a breakdown: the totals for one author, day or file.
type Group struct {
	Key     string
	Added   int
	Removed int
	Commits int
	Files   int // distinct file paths
}

// NetGrowth is added minus removed lines.
func (g Group) NetGrowth() int { return g.Added - g.Removed }

// Stats is the result of a query.
type Stats struct {
	Repository   string
	Authors      []string
	Period       string
	Branch       string
	Since        time.Time
	Until        time.Time
	Added        int
	Removed      int
	Commits      int
	FilesChanged int // distinct file paths
	GroupBy      string
	Groups       []Group // set when the query has a GroupBy
}

// NetGrowth is added minus removed lines.
func (s Stats) NetGrowth() int { return s.Added - s.Removed }
