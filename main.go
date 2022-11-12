package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var inPath *string

func main() {
	var packageLockJSON PackageLockJSON
	err := ReadPackageLockJSON(*inPath, &packageLockJSON)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Println(packageLockJSON)
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
