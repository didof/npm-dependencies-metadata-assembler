package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

var inPath, outPath *string
var dry *bool

type Payload struct {
	PackageLockJSON string                        `json:"package-lock.json"`
	Packages        map[string]ResolvedDependency `json:"packages"`
}

// TODO ignore file:

func main() {
	if len(*inPath) == 0 {
		if path, err := exec.LookPath("npm"); err != nil {
			log.Fatal("impossible continue")
		} else {
			if stats, err := os.Stat("package.json"); err != nil {
				log.Fatal("package.json missing")
			} else if stats.IsDir() {
				log.Fatal("package.json is not a file")
			}

			fmt.Println("package-lock.json not found. Producing it.")

			cmd := exec.Command(path, "install", "--package-lock-only", "--no-audit")
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}

			*inPath = "package-lock.json"
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		fmt.Printf("Received %v\n", sig)
		cancel()
	}()

	pCh, encCh, err := ReadPackageLockJSON(*inPath)
	if err != nil {
		log.Fatal(err)
	}

	payload := new(Payload)
	payload.Packages = make(map[string]ResolvedDependency)

jobs:
	for i := 0; i < 2; i++ {
		select {
		case <-ctx.Done():
			cancel()
			break jobs
		case p := <-pCh:
			defer func() { pCh = nil }()
			depsCh := p.DependenciesGenerator(ctx)
			workersChs := p.ResolveDependencies(ctx, runtime.NumCPU(), depsCh)
			resolvedCh := p.ReadResolvers(ctx, workersChs...)

			for dep := range resolvedCh {
				payload.Packages[dep.Name] = dep
			}
			fmt.Println("all dependencies have been resolved")
		case enc := <-encCh:
			defer func() { encCh = nil }()
			payload.PackageLockJSON = enc
			fmt.Println("package-lock.json has been encoded")
		}
	}

	b, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}

	if *dry {
		fmt.Println(string(b))
	}

	if len(*outPath) > 0 {
		err := os.WriteFile(*outPath, b, 0666)
		if err != nil {
			log.Fatal(err)
		}
	}

	os.Exit(0)
}

func init() {
	flag.Usage = usage
	inPath = flag.String("i", "", "The path to the package-lock.json file.")
	outPath = flag.String("o", "payload.json", "The path where to write the payload.")
	dry = flag.Bool("dry", false, "When dry is enabled the payload is printed to screen instead of beeing sent.")
	flag.Parse()
}

func usage() {
	fmt.Println("cli")
	flag.PrintDefaults()
	os.Exit(2)
}
