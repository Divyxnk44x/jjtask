package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents .jj-workspaces.yaml
type Config struct {
	Repos []Repo `yaml:"repos"`
}

// Repo represents a single repo in the config
type Repo struct {
	Path string `yaml:"path"`
	Name string `yaml:"name"`
}

// FindConfig locates .jj-workspaces.yaml by traversing up from cwd
func FindConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir != "/" {
		configPath := filepath.Join(dir, ".jj-workspaces.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", nil
}

// Load reads the workspace config
func Load() (*Config, string, error) {
	configPath, err := FindConfig()
	if err != nil {
		return nil, "", err
	}
	if configPath == "" {
		return nil, "", nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, "", err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, "", err
	}

	root := filepath.Dir(configPath)
	return &cfg, root, nil
}

// IsMultiRepo returns true if multi-repo config exists
func IsMultiRepo() bool {
	cfg, _, _ := Load()
	return cfg != nil && len(cfg.Repos) > 1
}

// GetRepos returns list of repo paths
func GetRepos() ([]Repo, string, error) {
	cfg, root, err := Load()
	if err != nil {
		return nil, "", err
	}
	if cfg == nil {
		return []Repo{{Path: ".", Name: "workspace"}}, "", nil
	}
	return cfg.Repos, root, nil
}

// ResolveRepoPath resolves a repo path relative to workspace root
func ResolveRepoPath(repo Repo, workspaceRoot string) string {
	if repo.Path == "." {
		return workspaceRoot
	}
	if filepath.IsAbs(repo.Path) {
		return repo.Path
	}
	return filepath.Join(workspaceRoot, repo.Path)
}

// DisplayName returns the display name for a repo
func DisplayName(repo Repo) string {
	if repo.Name != "" {
		return repo.Name
	}
	if repo.Path == "." {
		return "workspace"
	}
	return repo.Path
}

// RelativePath computes relative path from cwd to target
func RelativePath(target string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return target
	}
	rel, err := filepath.Rel(cwd, target)
	if err != nil {
		return target
	}
	if rel == "." {
		return "."
	}
	if !strings.HasPrefix(rel, "..") {
		return "./" + rel
	}
	return rel
}

// ContextHint returns context hint for multi-repo or subdirectory usage
func ContextHint() string {
	cfg, workspaceRoot, err := Load()
	if err != nil || cfg == nil {
		return ""
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Resolve symlinks for comparison
	realCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		realCwd = cwd
	}
	realRoot, err := filepath.EvalSymlinks(workspaceRoot)
	if err != nil {
		realRoot = workspaceRoot
	}

	isMulti := len(cfg.Repos) > 1
	inSubdir := realCwd != realRoot

	if !isMulti && !inSubdir {
		return ""
	}

	// Find which repo we're in
	var currentRepo string
	for _, repo := range cfg.Repos {
		repoPath := ResolveRepoPath(repo, workspaceRoot)
		realRepo, err := filepath.EvalSymlinks(repoPath)
		if err != nil {
			realRepo = repoPath
		}
		if strings.HasPrefix(realCwd, realRepo) {
			currentRepo = DisplayName(repo)
			break
		}
	}

	cwdRel := "."
	if realCwd != realRoot {
		rel, err := filepath.Rel(realRoot, realCwd)
		if err == nil {
			cwdRel = rel
		}
	}

	if cwdRel == "." {
		return fmt.Sprintf("cwd: . | repo: %s", currentRepo)
	}

	// Compute relative path to workspace root
	depth := strings.Count(cwdRel, string(filepath.Separator))
	rootRel := ".."
	for range depth {
		rootRel = "../" + rootRel
	}

	return fmt.Sprintf("cwd: %s | repo: %s | workspace: %s", cwdRel, currentRepo, rootRel)
}
