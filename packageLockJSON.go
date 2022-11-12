package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type PackageLockJSON struct {
	Name         string                          `json:"name"`
	Version      string                          `json:"version"`
	Dependencies map[string]UnresolvedDependency `json:"dependencies"`
}

func (p *PackageLockJSON) DependenciesGenerator(ctx context.Context) chan UnresolvedNamedDepedency {
	out := make(chan UnresolvedNamedDepedency)

	go func() {
		defer close(out)

		for name, data := range p.Dependencies {
			dep := UnresolvedNamedDepedency{Name: name, Version: data.Version, Resolved: data.Resolved}
			out <- dep
		}

	}()

	return out
}

func (p *PackageLockJSON) ResolveDependencies(ctx context.Context, n int, in <-chan UnresolvedNamedDepedency) []chan ResolvedDependency {
	chs := make([]chan ResolvedDependency, n)

	for i := 0; i < n; i++ {
		chs[i] = resolve(ctx, in)
	}

	return chs
}

func (p *PackageLockJSON) ReadResolvers(ctx context.Context, ins ...chan ResolvedDependency) chan ResolvedDependency {
	out := make(chan ResolvedDependency)

	var wg sync.WaitGroup
	wg.Add(len(ins))

	for _, in := range ins {
		go func(ch <-chan ResolvedDependency) {
			defer wg.Done()

		loop:
			for {
				select {
				case <-ctx.Done():
					break loop
				case w, ok := <-ch:
					if !ok {
						break loop
					}
					out <- w
				}
			}
		}(in)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

type Response struct {
	Dist struct {
		Shasum string `json:"shasum"`
	} `json:"dist"`
}

func resolve(ctx context.Context, ch <-chan UnresolvedNamedDepedency) chan ResolvedDependency {
	out := make(chan ResolvedDependency)

	go func() {
		defer close(out)

	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case dep, ok := <-ch:
				if !ok {
					break loop
				}

				url := strings.Split(dep.Resolved, "/-/")[0]
				url += fmt.Sprintf("/%s", dep.Version)

				req, err := http.NewRequest(http.MethodGet, url, nil)
				if err != nil {
					log.Fatal(err)
				}

				req = req.WithContext(ctx)

				client := new(http.Client)

				res, err := client.Do(req)
				if err != nil {
					// TODO return err as first citizen
					log.Fatal(err)
				}
				defer res.Body.Close()

				d := json.NewDecoder(res.Body)
				v := new(Response)
				err = d.Decode(v)
				if err != nil {
					log.Fatal(err)
				}

				out <- ResolvedDependency{
					Name:    dep.Name,
					Version: dep.Version,
					Shasum:  v.Dist.Shasum}
			}
		}
	}()

	return out
}

type UnresolvedDependency struct {
	Version  string `json:"version"`
	Resolved string `json:"resolved"`
}

type UnresolvedNamedDepedency struct {
	Name, Version, Resolved string
}

type ResolvedDependency struct {
	Name, Version, Shasum string
}

func ReadPackageLockJSON(path string, packageLockJSON *PackageLockJSON) error {
	if stats, err := os.Stat(*inPath); err != nil {
		return fmt.Errorf("the file %s does not exists", path)
	} else if stats.IsDir() {
		return fmt.Errorf("the file %s is a dir", path)
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		err = json.Unmarshal(data, packageLockJSON)
		if err != nil {
			return err
		}
	}
	return nil
}
