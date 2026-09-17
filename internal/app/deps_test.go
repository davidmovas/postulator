package app_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/davidmovas/postulator"

const (
	kernelTree      = "internal/kernel"
	domainTree      = "internal/domain"
	applicationTree = "internal/application"
	adaptersTree    = "internal/adapters"
	runtimeTree     = "internal/runtime"
	transportTree   = "internal/transport"
	appTree         = "internal/app"
	pagingPackage   = "internal/kernel/paging"
)

const squirrel = "github.com/Masterminds/squirrel"

func repoRoot(t *testing.T) string {
	t.Helper()

	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}

	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		t.Fatal("the dependency rule test must run inside the module")
	}
	return filepath.Dir(gomod)
}

func treeExists(t *testing.T, tree string) bool {
	t.Helper()

	info, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(tree)))
	return err == nil && info.IsDir()
}

func transitiveDeps(t *testing.T, tree string) []string {
	t.Helper()

	cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", "./"+tree+"/...")
	cmd.Dir = repoRoot(t)

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps ./%s/...: %v", tree, err)
	}
	return strings.Fields(string(out))
}

func directImports(t *testing.T) map[string][]string {
	t.Helper()

	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", "./...")
	cmd.Dir = repoRoot(t)

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list ./...: %v", err)
	}

	imports := make(map[string][]string)
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		imports[fields[0]] = fields[1:]
	}
	return imports
}

func isThirdParty(path string) bool {
	if path == modulePath || strings.HasPrefix(path, modulePath+"/") {
		return false
	}
	first, _, _ := strings.Cut(path, "/")
	return strings.Contains(first, ".")
}

func ownPackage(path, tree string) bool {
	return strings.HasPrefix(path, modulePath+"/"+tree+"/") || path == modulePath+"/"+tree
}

func TestLayersDoNotReachUpwards(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		tree      string
		forbidden []string
	}{
		{
			name:      "kernel is the bottom layer",
			tree:      kernelTree,
			forbidden: []string{domainTree, applicationTree, adaptersTree, runtimeTree, transportTree, appTree},
		},
		{
			name:      "domain is pure",
			tree:      domainTree,
			forbidden: []string{applicationTree, adaptersTree, runtimeTree, transportTree, appTree},
		},
		{
			name:      "application knows only domain and kernel",
			tree:      applicationTree,
			forbidden: []string{adaptersTree, runtimeTree, transportTree, appTree},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if !treeExists(t, tc.tree) {
				t.Skipf("%s does not exist yet", tc.tree)
			}

			for _, dep := range transitiveDeps(t, tc.tree) {
				for _, forbidden := range tc.forbidden {
					if ownPackage(dep, forbidden) {
						t.Errorf("%s depends on %s", tc.tree, dep)
					}
				}
			}
		})
	}
}

func TestDomainCarriesNoThirdPartyDependencies(t *testing.T) {
	t.Parallel()

	if !treeExists(t, domainTree) {
		t.Skipf("%s does not exist yet", domainTree)
	}

	allowed := map[string]struct{}{"golang.org/x/net/html": {}, "golang.org/x/net/html/atom": {}}
	for _, dep := range transitiveDeps(t, domainTree) {
		if !isThirdParty(dep) {
			continue
		}
		if _, ok := allowed[dep]; !ok {
			t.Errorf("domain pulls in the third-party package %s", dep)
		}
	}
}

func TestOnlyPagingUsesTheQueryBuilder(t *testing.T) {
	t.Parallel()

	for pkg, imports := range directImports(t) {
		if !ownPackage(pkg, kernelTree) && !ownPackage(pkg, domainTree) && !ownPackage(pkg, applicationTree) {
			continue
		}
		if ownPackage(pkg, pagingPackage) {
			continue
		}
		for _, imported := range imports {
			if imported == squirrel {
				t.Errorf("%s imports the query builder; only %s may", pkg, pagingPackage)
			}
		}
	}
}

func TestOnlyTheCompositionRootImportsTransport(t *testing.T) {
	t.Parallel()

	for pkg, imports := range directImports(t) {
		if ownPackage(pkg, appTree) || ownPackage(pkg, transportTree) {
			continue
		}
		for _, imported := range imports {
			if ownPackage(imported, transportTree) {
				t.Errorf("%s imports %s; only %s may", pkg, imported, appTree)
			}
		}
	}
}
