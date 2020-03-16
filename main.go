package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	// Parse config.
	if len(os.Args) != 2 {
		log.Fatal("please specify the path to a config file, an example config is available at https://github.com/beefsack/git-mirror/blob/master/example-config.toml")
	}
	cfg, repos, err := parseConfig(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(cfg.BasePath, 0755); err != nil {
		log.Fatalf("failed to create %s, %s", cfg.BasePath, err)
	}

	// Run background threads to keep mirrors up to date.
	wg := sync.WaitGroup{}
	for _, r := range repos {
		wg.Add(1)
		go func(r repo) {
			defer wg.Done()
			for {
				log.Printf("updating %s", r.Name)
				if err := mirror(cfg, r); err != nil {
					log.Printf("error updating %s, %s", r.Name, err)
				} else if r.Target != "" {
					log.Printf("updated %s (pushed to %s)", r.Name, r.Target)
				} else {
					log.Printf("updated %s", r.Name)
				}
				time.Sleep(r.Interval.Duration)
			}
		}(r)
	}

	// Run HTTP server to serve mirrors.
	if cfg.ServeMirror {
		handler := http.FileServer(http.Dir(cfg.BasePath))
		if cfg.AutoClone {
			handler = handleGitClone(cfg, handler)
		}
		http.Handle("/", handler)
		log.Printf("starting web server on %s", cfg.ListenAddr)
		if err := http.ListenAndServe(cfg.ListenAddr, nil); err != nil {
			log.Fatalf("failed to start server, %s", err)
		}
	}

	wg.Wait()
}

func handleGitClone(cfg config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rpstr := r.URL.String()[1:]

		if strings.Index(rpstr, ".git/info/") > -1 || strings.HasSuffix(rpstr, ".git") {
			rpstr = rpstr[:strings.Index(rpstr, ".git")+4]
			log.Print("Cloning: ", rpstr)
			rp := repo{rpstr, "https://" + rpstr, "", duration{}}
			if err := mirror(cfg, rp); err != nil {
				log.Print("Clone error: ", err, rp, r)
			}
		}

		next.ServeHTTP(w, r)
	})
}
