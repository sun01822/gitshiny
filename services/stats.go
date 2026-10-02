// Package services contains the statistics engine. It knows nothing about the
// CLI or TUI, so any interface can reuse it.
package services

import "github.com/USERNAME/gitshiny/domain"

// CommitSource provides commits for a query (implemented by repositories.Git).
type CommitSource interface {
	Log(domain.Query) ([]domain.Commit, error)
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
	}
	files := make(map[string]struct{})
	for _, c := range commits {
		for _, f := range c.Files {
			s.Added += f.Added
			s.Removed += f.Removed
			files[f.Path] = struct{}{}
		}
	}
	s.FilesChanged = len(files)
	return s, nil
}
