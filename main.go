package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var inPath *string

type Payload struct {
	PackageLockJSON string
	Dependencies    map[string]ResolvedDependency
}

func main() {
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
	payload.Dependencies = make(map[string]ResolvedDependency)

jobs:
	for i := 0; i < 2; i++ {
		select {
		case <-ctx.Done():
			cancel()
			break jobs
		case p := <-pCh:
			depsCh := p.DependenciesGenerator(ctx)
			workersChs := p.ResolveDependencies(ctx, runtime.NumCPU(), depsCh)
			resolvedCh := p.ReadResolvers(ctx, workersChs...)

			for dep := range resolvedCh {
				payload.Dependencies[dep.Name] = dep
			}
			fmt.Println("all dependencies have been resolved")
		case enc := <-encCh:
			payload.PackageLockJSON = enc
			fmt.Println("package-lock.json has been encoded")
		}
	}
}

func init() {
	flag.Usage = usage
	inPath = flag.String("i", "./package-lock.json", "The path to the package-lock.json file.")

	flag.Parse()
}

func usage() {
	fmt.Println("cli")
	flag.PrintDefaults()
	os.Exit(2)
}
