// Package services contains the statistics engine. It knows nothing about the
// CLI or TUI, so any interface can reuse it.
package services

import (
	"sort"

	"github.com/sun01822/gitshiny/domain"
)

// CommitSource provides commits for a query (implemented by repositories.Git).
type CommitSource interface {
	Log(domain.Query) ([]domain.Commit, error)
}

// tally collects one total: lines, commits and the distinct files behind it.
type tally struct {
	added, removed, commits int
	files                   map[string]struct{}
}

// Calculate aggregates the commits matching q into Stats.
func Calculate(src CommitSource, q domain.Query) (domain.Stats, error) {
	commits, err := src.Log(q)
	if err != nil {
		return domain.Stats{}, err
	}
	s := domain.Stats{
		Authors: q.Authors,
		Period:  q.Period,
		Branch:  q.Branch,
		Since:   q.Since,
		Until:   q.Until,
		Commits: len(commits),
		GroupBy: q.GroupBy,
	}
	files := make(map[string]struct{})
	groups := make(map[string]*tally)
	for _, c := range commits {
		counted := make(map[string]bool) // groups this commit already counts toward
		for _, f := range c.Files {
			s.Added += f.Added
			s.Removed += f.Removed
			files[f.Path] = struct{}{}
			if q.GroupBy == "" {
				continue
			}
			key := groupKey(q.GroupBy, c, f)
			g := groups[key]
			if g == nil {
				g = &tally{files: make(map[string]struct{})}
				groups[key] = g
			}
			g.added += f.Added
			g.removed += f.Removed
			g.files[f.Path] = struct{}{}
			if !counted[key] {
				counted[key] = true
				g.commits++
			}
		}
	}
	s.FilesChanged = len(files)
	for key, g := range groups {
		s.Groups = append(s.Groups, domain.Group{
			Key: key, Added: g.added, Removed: g.removed, Commits: g.commits, Files: len(g.files),
		})
	}
	// Days read oldest first; authors and files put the most changed first.
	sort.Slice(s.Groups, func(i, j int) bool {
		a, b := s.Groups[i], s.Groups[j]
		if churn := a.Added + a.Removed - b.Added - b.Removed; q.GroupBy != "day" && churn != 0 {
			return churn > 0
		}
		return a.Key < b.Key
	})
	return s, nil
}

func groupKey(by string, c domain.Commit, f domain.FileChange) string {
	switch by {
	case "author":
		return c.Author
	case "day":
		return c.Time.Format("2006-01-02")
	}
	return f.Path
}
