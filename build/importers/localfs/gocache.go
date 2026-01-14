// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package localfs

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"
	"github.com/gx-org/gx/build/importers"
)

const goModCache = "GOMODCACHE"

func modCachePath() (string, error) {
	cmd := exec.Command("go", "env")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	for line := range strings.Lines(string(out)) {
		spLine := strings.Split(line, "=")
		if len(spLine) != 2 {
			continue
		}
		if spLine[0] != goModCache {
			continue
		}
		cachePath := strings.TrimSpace(spLine[1])
		cachePath = strings.TrimPrefix(cachePath, "'")
		cachePath = strings.TrimSuffix(cachePath, "'")
		return cachePath, nil
	}
	return "", fmt.Errorf("Go variable environment %s not found", goModCache)
}

func (imp *Importer) moduleOSPath(dep *module.Version) (string, error) {
	dir, file := filepath.Split(dep.Path)
	osPath := filepath.Join(imp.goCachePath, dir, fmt.Sprintf("%s@%s", file, dep.Version))
	dirStat, err := os.Stat(osPath)
	if err != nil {
		return "", fmt.Errorf("cannot find GX module path at %s: %v", dirStat, err)
	}
	if !dirStat.IsDir() {
		return "", fmt.Errorf("%s not a directory", osPath)
	}
	return osPath, nil
}

func (imp *Importer) importFromGoCache(bld importers.Builder, importPath string, dep *module.Version) (importers.Package, error) {
	osPath, err := imp.moduleOSPath(dep)
	if err != nil {
		return nil, err
	}
	pkgPath := importPath[len(dep.Path)+1:]
	fullPath := filepath.Join(osPath, pkgPath)
	dirStat, err := os.Stat(fullPath)
	if !dirStat.IsDir() || err != nil {
		return nil, fmt.Errorf("cannot find GX package %s in module %s", pkgPath, dep.String())
	}
	return ImportAt(bld, os.DirFS(osPath).(fs.ReadDirFS), importPath, pkgPath)
}
