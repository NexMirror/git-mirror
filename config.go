package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type duration struct {
	time.Duration
}

type config struct {
	ServeMirror bool
	AutoClone   bool
	ListenAddr  string
	Interval    duration
	BasePath    string
	KeyPath     string
	Repo        []repo
}

type repo struct {
	Name     string
	Origin   string
	Target   string
	Interval duration
}

func (d *duration) UnmarshalText(text []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(text))
	return
}

func parseConfig(filename string) (cfg config, repos map[string]repo, err error) {
	// Parse the raw TOML file.
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("unable to read config file %s, %s", filename, err)
		return
	}
	var meta toml.MetaData
	if meta, err = toml.Decode(string(raw), &cfg); err != nil {
		err = fmt.Errorf("unable to load config %s, %s", filename, err)
		return
	}

	// Set defaults if required.
	if !meta.IsDefined("ServeMirror") {
		cfg.ServeMirror = true
	}
	if !meta.IsDefined("AutoClone") {
		cfg.AutoClone = false
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.Interval.Duration == 0 {
		cfg.Interval.Duration = 15 * time.Minute
	}
	userHome := "/root"
	if user, err := user.Current(); err == nil {
		userHome = user.HomeDir
	}
	if cfg.BasePath == "" {
		cfg.BasePath = "."
	}
	if cfg.BasePath[0] == '~' {
		cfg.BasePath = userHome + cfg.BasePath[1:]
	} else if cfg.BasePath, err = filepath.Abs(cfg.BasePath); err != nil {
		err = fmt.Errorf("unable to get absolute path to base path, %s", err)
		return
	}
	if cfg.KeyPath[0] == '~' {
		cfg.KeyPath = userHome + cfg.KeyPath[1:]
	} else if cfg.KeyPath, err = filepath.Abs(cfg.KeyPath); err != nil {
		err = fmt.Errorf("unable to get absolute path to key path, %s", err)
		return
	}

	if _, err = os.Stat(cfg.KeyPath); os.IsNotExist(err) {
		err = fmt.Errorf("%s does not exists", cfg.KeyPath)
		return
	} else if err = ioutil.WriteFile("git-ssh", []byte("ssh -i "+cfg.KeyPath+" \"$@\""), 0777); err != nil {
		err = fmt.Errorf("failed to create git-ssh")
		return
	}

	// Fetch repos, injecting default values where needed.
	if cfg.Repo == nil || len(cfg.Repo) == 0 {
		err = fmt.Errorf("no repos found in config %s, please define repos under [[repo]] sections", filename)
		return
	}
	repos = map[string]repo{}
	for i, r := range cfg.Repo {
		if r.Origin == "" {
			err = fmt.Errorf("Origin required for repo %d in config %s", i+1, filename)
			return
		}

		// Generate a name if there isn't one already
		if r.Name == "" {
			origin := r.Origin
			if strings.HasPrefix(r.Origin, "hg::") {
				origin = origin[4:]
			}
			if u, err := url.Parse(origin); err == nil && u.Scheme != "" {
				r.Name = u.Host + u.Path
			} else {
				parts := strings.Split(origin, "@")
				if l := len(parts); l > 0 {
					r.Name = strings.Replace(parts[l-1], ":", "/", -1)
				}
			}
		}
		if r.Name == "" {
			err = fmt.Errorf("Could not generate name for Origin %s in config %s, please manually specify a Name", r.Origin, filename)
		}
		if _, ok := repos[r.Name]; ok {
			err = fmt.Errorf("Multiple repos with name %s in config %s", r.Name, filename)
			return
		}

		if r.Interval.Duration == 0 {
			r.Interval.Duration = cfg.Interval.Duration
		}
		repos[r.Name] = r
	}
	return
}
