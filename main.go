package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/akamensky/argparse"
	"github.com/goccy/go-graphviz"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func main() {
	parser := argparse.NewParser("print", "Prints provided string to stdout")
	dir := parser.String("d", "dir", &argparse.Options{
		Required: false,
		Help:     "working dir for packages.Load",
		Default:  "",
	})
	packageNames := parser.StringList("p", "pkg", &argparse.Options{
		Required: false,
		Help:     "package to parse",
		Default:  []string{},
	})
	err := parser.Parse(os.Args)
	if err != nil {
		log.Fatal(parser.Usage(err))
	}

	program, ssapackages := callGraph(*dir, *packageNames)
	program.Build()

	graph := cha.CallGraph(program)

	// render graph as Graphviz
	ssaPkgSet := make(map[*ssa.Package]struct{}, len(ssapackages))
	for _, ssapkg := range ssapackages {
		ssaPkgSet[ssapkg] = struct{}{}
	}
	gv := toGraphviz(graph, ssaPkgSet)
	fmt.Println(gv.String())
}

func callGraph(dir string, packageNames []string) (*ssa.Program, []*ssa.Package) {
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

func toGraphviz(cgraph *callgraph.Graph, ssaPkgSet map[*ssa.Package]struct{}) *bytes.Buffer {
	graph := graphviz.New()
	g, err := graph.Graph()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := g.Close(); err != nil {
			log.Fatal(err)
		}
		graph.Close()
	}()

	// callgraph.GraphVisitEdges()
	err = traverseExported(cgraph, ssaPkgSet, func(e *callgraph.Edge) error {
		// child
		if e.Callee.ID == e.Caller.ID {
			return nil
		}
		child, err := g.CreateNode(e.Caller.String())
		if err != nil {
			log.Fatal(err)
		}
		if _, ok := ssaPkgSet[e.Caller.Func.Pkg]; ok {
			child.SetLabel(e.Caller.Func.Name())
		}

		// // parent
		parent, err := g.CreateNode(e.Callee.String())
		if err != nil {
			log.Fatal(err)
		}
		if _, ok := ssaPkgSet[e.Callee.Func.Pkg]; ok {
			parent.SetLabel(e.Callee.Func.Name())
		}

		// // edge
		_, err = g.CreateEdge("e", child, parent)
		if err != nil {
			log.Fatal(err)
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	if err := graph.Render(g, "dot", &buf); err != nil {
		log.Fatal(err)
	}

	return &buf
}

func traverseExported(g *callgraph.Graph, ssaPkgSet map[*ssa.Package]struct{}, edgeWorker func(*callgraph.Edge) error) error {
	exportedNativeNames := make(map[string]struct{})
	for pkg := range ssaPkgSet {
		for _, name := range pkg.Pkg.Scope().Names() {
			if isExported(name) {
				exportedNativeNames[name] = struct{}{}
			}
		}
	}

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
