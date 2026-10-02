package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/USERNAME/gitshiny/domain"
	"github.com/USERNAME/gitshiny/utils"
)

const rule = "========================================"

type jsonOutput struct {
	Repository   string `json:"repository"`
	Author       string `json:"author"`
	Branch       string `json:"branch"`
	Period       string `json:"period"`
	Since        string `json:"since,omitempty"`
	Until        string `json:"until,omitempty"`
	Added        int    `json:"added"`
	Removed      int    `json:"removed"`
	NetGrowth    int    `json:"net_growth"`
	Commits      int    `json:"commits"`
	FilesChanged int    `json:"files_changed"`
}

func authorLabel(s domain.Stats) string {
	if len(s.Authors) == 0 {
		return "All authors"
	}
	return strings.Join(s.Authors, ", ")
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func branchLabel(s domain.Stats) string {
	if s.Branch == "" {
		return "all branches"
	}
	return s.Branch
}

// Render writes s in the given format: text, json or csv.
func Render(w io.Writer, s domain.Stats, format string) error {
	switch format {
	case "text":
		return renderText(w, s)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(jsonOutput{
			Repository: s.Repository, Author: authorLabel(s), Branch: branchLabel(s),
			Period: s.Period, Since: fmtTime(s.Since), Until: fmtTime(s.Until),
			Added: s.Added, Removed: s.Removed, NetGrowth: s.NetGrowth(),
			Commits: s.Commits, FilesChanged: s.FilesChanged,
		})
	case "csv":
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"repository", "author", "branch", "period", "since", "until",
			"added", "removed", "net_growth", "commits", "files_changed"})
		_ = cw.Write([]string{s.Repository, authorLabel(s), branchLabel(s), s.Period,
			fmtTime(s.Since), fmtTime(s.Until),
			strconv.Itoa(s.Added), strconv.Itoa(s.Removed), strconv.Itoa(s.NetGrowth()),
			strconv.Itoa(s.Commits), strconv.Itoa(s.FilesChanged)})
		cw.Flush()
		return cw.Error()
	}
	return fmt.Errorf("unknown format %q (use text, json or csv)", format)
}

func renderText(w io.Writer, s domain.Stats) error {
	since, until := fmtTime(s.Since), fmtTime(s.Until)
	if since == "" {
		since = "beginning"
	}
	if until == "" {
		until = "now"
	}
	_, err := fmt.Fprintf(w, `%[1]s
             Git Statistics
%[1]s

Repository : %[2]s
Branch     : %[3]s
Author     : %[4]s
Period     : %[5]s
Since      : %[6]s
Until      : %[7]s

Added lines    : %[8]s
Removed lines  : %[9]s
Net growth     : %[10]s
Commits        : %[11]s
Files changed  : %[12]s

%[1]s
`, rule, s.Repository, branchLabel(s), authorLabel(s), s.Period, since, until,
		utils.Commas(s.Added), utils.Commas(s.Removed), utils.Signed(s.NetGrowth()),
		utils.Commas(s.Commits), utils.Commas(s.FilesChanged))
	return err
}
