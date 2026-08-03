package generator

import (
	"go/types"
	"path/filepath"
	"sync"

	"golang.org/x/tools/go/packages"
)

type packageTypeInfo struct {
	once     sync.Once
	pkg      *types.Package
	complete bool
}

func (goPackage *GoPackage) loadTypePackage() (*types.Package, bool) {
	if goPackage == nil {
		return nil, false
	}
	goPackage.typeInfo.once.Do(func() {
		if len(goPackage.SourceFiles) == 0 {
			return
		}

		sourceFile, err := filepath.Abs(goPackage.SourceFiles[0])
		if err != nil {
			return
		}
		loaded, err := packages.Load(&packages.Config{
			Dir:  goPackage.Dir,
			Mode: packages.LoadTypes,
		}, "file="+sourceFile)
		if err != nil {
			return
		}

		for _, candidate := range loaded {
			if candidate == nil || candidate.Types == nil || candidate.Name != goPackage.Name {
				continue
			}
			goPackage.typeInfo.pkg = candidate.Types
			goPackage.typeInfo.complete = !candidate.IllTyped && len(candidate.Errors) == 0
			return
		}
	})
	return goPackage.typeInfo.pkg, goPackage.typeInfo.complete
}
