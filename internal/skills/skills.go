// Package skills discovers Agent Skills (SKILL.md files with YAML frontmatter).
package skills

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	maxNameLen        = 64
	maxDescriptionLen = 1024

	metaAutoInvoke = "opencode/autoinvoke"
)

var nameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Skill struct {
	Name        string
	Description string
	License     string
	Metadata    map[string]string
	// Body is the markdown after the frontmatter.
	Body string
	Path string
}

// AutoInvoke reports whether the skill is listed in model-facing guidance.
func (s Skill) AutoInvoke() bool {
	return !strings.EqualFold(s.Metadata[metaAutoInvoke], "false")
}

type frontmatter struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	License     string         `yaml:"license"`
	Metadata    map[string]any `yaml:"metadata"`
}

// Parse reads a SKILL.md document. dirName must equal the skill name.
func Parse(data []byte, dirName string) (Skill, error) {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return Skill{}, fmt.Errorf("missing YAML frontmatter")
	}
	rest := data[len("---\n"):]
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return Skill{}, fmt.Errorf("unterminated YAML frontmatter")
	}

	var fm frontmatter
	if err := yaml.Unmarshal(rest[:end], &fm); err != nil {
		return Skill{}, fmt.Errorf("invalid frontmatter: %w", err)
	}
	if err := validate(fm, dirName); err != nil {
		return Skill{}, err
	}

	body := rest[end+len("\n---"):]
	if i := bytes.IndexByte(body, '\n'); i >= 0 {
		body = body[i+1:]
	} else {
		body = nil
	}

	meta := make(map[string]string, len(fm.Metadata))
	for k, v := range fm.Metadata {
		meta[k] = fmt.Sprint(v)
	}

	return Skill{
		Name:        fm.Name,
		Description: strings.TrimSpace(fm.Description),
		License:     fm.License,
		Metadata:    meta,
		Body:        strings.TrimSpace(string(body)),
	}, nil
}

func validate(fm frontmatter, dirName string) error {
	if fm.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(fm.Name) > maxNameLen || !nameRe.MatchString(fm.Name) {
		return fmt.Errorf("invalid name %q: use 1-%d lowercase alphanumeric characters with single hyphens", fm.Name, maxNameLen)
	}
	if fm.Name != dirName {
		return fmt.Errorf("name %q must match directory name %q", fm.Name, dirName)
	}
	desc := strings.TrimSpace(fm.Description)
	if desc == "" || len(desc) > maxDescriptionLen {
		return fmt.Errorf("description must be 1-%d characters", maxDescriptionLen)
	}
	return nil
}

// Discover finds skills for a working directory. Project locations (walking up
// to the git root) take precedence over global ones; the first skill with a
// given name wins. Invalid skills are skipped and reported in the returned warnings.
func Discover(cwd, home string) ([]Skill, []string) {
	var dirs []string
	for dir := cwd; ; dir = filepath.Dir(dir) {
		dirs = append(dirs,
			filepath.Join(dir, ".opencode", "skills"),
			filepath.Join(dir, ".claude", "skills"),
			filepath.Join(dir, ".agents", "skills"),
		)
		if isGitRoot(dir) || filepath.Dir(dir) == dir {
			break
		}
	}
	if home != "" {
		dirs = append(dirs,
			filepath.Join(home, ".config", "opencode", "skills"),
			filepath.Join(home, ".claude", "skills"),
			filepath.Join(home, ".agents", "skills"),
		)
	}

	seen := map[string]bool{}
	var found []Skill
	var warnings []string
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			path := filepath.Join(dir, e.Name(), "SKILL.md")
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			skill, err := Parse(data, e.Name())
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s: %v", path, err))
				continue
			}
			if seen[skill.Name] {
				continue
			}
			seen[skill.Name] = true
			skill.Path = path
			found = append(found, skill)
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Name < found[j].Name })
	return found, warnings
}

func isGitRoot(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}
