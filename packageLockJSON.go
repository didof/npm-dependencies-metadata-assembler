package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type PackageLockJSON struct {
	Name         string                          `json:"name"`
	Version      string                          `json:"version"`
	Dependencies map[string]UnresolvedDependency `json:"dependencies"`
}

type Dependency struct {
	Version string `json:"version"`
}

type UnresolvedDependency struct {
	Dependency
	Resolved string `json:"resolved"`
}

type ResolvedDependency struct {
	Dependency
	Shasum string
}

func (d UnresolvedDependency) Resolve() {
	fmt.Println(d)
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
