package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func loadPackages(dir string, packageNames []string) (*ssa.Program, []*ssa.Package) {
	cfg := packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes |
			packages.NeedDeps |
			packages.NeedSyntax,
		Dir: dir,
	}

	pkgs, err := packages.Load(&cfg, packageNames...)
	if err != nil {
		log.Fatalf("error loading packages %v\n", err)
	}
	// log.Println(pkgs)
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}
	// AllPackages is super slow but includes all dependencies
	program, ssapackages := ssautil.Packages(pkgs, ssa.InstantiateGenerics)

	return program, ssapackages
}

func stdPackageNames() map[string]struct{} {
	stdPkgs, err := packages.Load(nil, "std")
	if err != nil {
		log.Fatalf("can't load standard library packages: %v", err)
	}
	stdPkgNames := make(map[string]struct{}, len(stdPkgs))
	for _, pkg := range stdPkgs {
		stdPkgNames[pkg.Name] = struct{}{}
	}
	return stdPkgNames
}

func traverseExported(g *callgraph.Graph, ssaPkgSet map[*ssa.Package]struct{}, edgeWorker func(*callgraph.Edge) error) error {
	exportedNativeNames := make(map[string]struct{})
	for pkg := range ssaPkgSet {
		for name, member := range pkg.Members {
			// fmt.Fprintf(os.Stderr, "name      = %v\n", name)
			if member == nil {
				fmt.Fprintln(os.Stderr, "*** NIL MEMBER ***")
				continue
			}
			if member.Object() != nil && member.Object().Exported() {
				fmt.Fprintf(os.Stderr, "name      = %v\n", name)
				fmt.Fprintf(os.Stderr, "objID     = %v\n", member.Object().Id())
				fmt.Fprintf(os.Stderr, "objName   = %v\n", member.Object().Name())
				// fmt.Fprintf(os.Stderr, "objParent = %v\n", member.Object().Parent())
				if member.Object().Pkg() != nil {
					fmt.Fprintf(os.Stderr, "objPkgName= %v\n", member.Object().Pkg().Name())
					fmt.Fprintf(os.Stderr, "objPkgPath= %v\n", member.Object().Pkg().Path())
				} else {
					fmt.Fprintln(os.Stderr, "*** NIL member.Object().Pkg() ***")
				}
				// fmt.Fprintf(os.Stderr, "objType   = %v\n", member.Object().Type())
				// fmt.Fprintf(os.Stderr, "type      = %v\n", member.Type())
				fmt.Fprintln(os.Stderr)
				exportedNativeNames[name] = struct{}{}
			}
		}
	}

	log.Println("filling the queue...")
	seen := make(map[int]bool)
	queue := make([]*callgraph.Edge, 0)
	for _, node := range g.Nodes {
		if node == nil {
			continue
		}
		if node.Func == nil {
			continue
		}
		if _, ok := exportedNativeNames[node.Func.Name()]; ok {
			seen[node.ID] = true
			for _, edge := range node.Out {
				if isExported(edge.Callee.Func.Name()) {
					queue = append(queue, edge)
				}
			}
		}
	}
	log.Println("traversing the graph...")
	for len(queue) > 0 {
		edge := queue[0]
		node := edge.Callee
		if !seen[node.ID] {
			seen[node.ID] = true
			err := edgeWorker(edge)
			if err != nil {
				return err
			}
			for _, edge := range node.Out {
				if isExported(edge.Callee.Func.Name()) {
					queue = append(queue, edge)
				}
			}
		}
		queue = queue[1:]
	}
	return nil
}

func isExported(name string) bool {
	return name[0] >= 'A' && name[0] <= 'Z'
}
