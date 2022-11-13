package main

import (
	"context"
	"encoding/base64"
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

func (p *PackageLockJSON) ResolveDependencies(ctx context.Context, n int, in <-chan UnresolvedNamedDepedency) []chan ResolveResult {
	chs := make([]chan ResolveResult, n)

	for i := 0; i < n; i++ {
		chs[i] = resolve(ctx, in)
	}

	return chs
}

func (p *PackageLockJSON) ReadResolvers(ctx context.Context, ins ...chan ResolveResult) chan ResolvedDependency {
	out := make(chan ResolvedDependency)

	var wg sync.WaitGroup
	wg.Add(len(ins))

	for _, in := range ins {
		go func(ch <-chan ResolveResult) {
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
					if w.Error != nil {
						// TODO Decice how to handle
						log.Fatal(w.Error)
					}

					out <- w.Value
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

type ResolveResult struct {
	Error error
	Value ResolvedDependency
}

func resolve(ctx context.Context, ch <-chan UnresolvedNamedDepedency) chan ResolveResult {
	out := make(chan ResolveResult)

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

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				if err != nil {
					r := new(ResolveResult)
					r.Error = err
					out <- *r
				}

				res, err := http.DefaultClient.Do(req)
				if err != nil {
					// TODO return err as first citizen
					log.Fatal(err)
				} else if res.StatusCode != http.StatusOK {
					log.Fatal("unexepcted response")
					return
				}
				defer res.Body.Close()

				v := new(Response)
				if err := json.NewDecoder(res.Body).Decode(v); err != nil {
					log.Fatal(err)
				} else {
					r := new(ResolveResult)
					r.Value = ResolvedDependency{
						Name:    dep.Name,
						Version: dep.Version,
						Shasum:  v.Dist.Shasum,
					}
					out <- *r
				}
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
	Name    string `json:"-"`
	Version string `json:"version"`
	Shasum  string `json:"shasum"`
}

func readFile(path string) ([]byte, error) {
	if stats, err := os.Stat(*inPath); err != nil {
		return nil, fmt.Errorf("the file %s does not exists", path)
	} else if stats.IsDir() {
		return nil, fmt.Errorf("the file %s is a dir", path)
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
}

func ReadPackageLockJSON(path string) (chan PackageLockJSON, chan string, error) {
	pCh := make(chan PackageLockJSON)
	encCh := make(chan string)

	var wg sync.WaitGroup
	wg.Add(2)

	data, err := readFile(path)
	if err != nil {
		return nil, nil, err
	}

	go func() {
		defer wg.Done()
		var p = new(PackageLockJSON)
		err = json.Unmarshal(data, p)
		if err != nil {
			log.Fatal(err)
		}

		pCh <- *p
	}()

	go func() {
		defer wg.Done()
		encCh <- base64.StdEncoding.EncodeToString(data)
	}()

	go func() {
		wg.Wait()
		close(pCh)
		close(encCh)
	}()

	return pCh, encCh, nil
}
