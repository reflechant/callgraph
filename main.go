package main

import (
	"fmt"
	"log"
	"os"

	"github.com/akamensky/argparse"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/ssa"
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

	// log.Println(stdPackageNames())
	log.Println("loading packages...")
	program, packages := loadPackages(*dir, *packageNames)
	program.Build()

	log.Println("calculating callgraph...")
	graph := cha.CallGraph(program)

	pkgSet := make(map[*ssa.Package]struct{}, len(packages))
	for _, pkg := range packages {
		pkgSet[pkg] = struct{}{}
	}
	// render graph as Graphviz
	log.Println("building Graphviz model...")
	gv := toGraphviz(graph, pkgSet)
	log.Println("rendering Graphviz...")
	fmt.Println(gv.String())
}
