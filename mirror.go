package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

func mirror(cfg config, r repo) error {
	var cmd *exec.Cmd
	repoPath := path.Join(cfg.BasePath, r.Name)

	if _, err := os.Stat(repoPath); err == nil {
		// Directory exists, update.
		cmd = exec.Command("git", "fetch", "-p", "origin")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to fetch origin in %s: %s", repoPath, err)
		}
	} else if os.IsNotExist(err) {
		// Clone
		parent := path.Dir(repoPath)
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for cloning %s, %s", repoPath, err)
		}
		cmd = exec.Command("git", "clone", "--mirror", r.Origin, repoPath)
		cmd.Dir = parent
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone %s: %s", r.Origin, err)
		}
	} else {
		return fmt.Errorf("failed to stat %s: %s", repoPath, err)
	}

	if cfg.ServeMirror {
		cmd = exec.Command("git", "update-server-info")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update-server-info for %s： %s", repoPath, err)
		}
	}

	if r.Target != "" {
		// git remote set-url --push origin
		cmd := exec.Command("git", "remote", "set-url", "--push", "origin", r.Target)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set target url to %s for %s: %s", r.Target, repoPath, err)
		}

		// git push --mirror --quiet
		cmd = exec.Command("git", "push", "--mirror", "--quiet")
		cmd.Dir = repoPath
		if cfg.KeyPath != "" {
			gitSshPath, err := filepath.Abs("git-ssh")
			if err != nil {
				return fmt.Errorf("unable to get absolute path to git-ssh: %s", err)
			}
			env := os.Environ()
			env = append(env, "GIT_SSH="+gitSshPath)
			cmd.Env = env
		}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to push to %s for %s: %s", r.Target, repoPath, err)
		}
	}

	return nil
}
