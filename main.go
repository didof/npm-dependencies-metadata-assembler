package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
)

var inPath *string

func main() {
	var p PackageLockJSON
	err := ReadPackageLockJSON(*inPath, &p)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // TODO read sigterm

	depsCh := p.DependenciesGenerator(ctx)
	workersChs := p.ResolveDependencies(ctx, runtime.NumCPU(), depsCh)
	resolvedCh := p.ReadResolvers(ctx, workersChs...)

	for dep := range resolvedCh {
		fmt.Println(dep)
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
