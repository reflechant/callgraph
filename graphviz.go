package main

import (
	"bytes"
	"log"

	"github.com/goccy/go-graphviz"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

func toGraphviz(cgraph *callgraph.Graph, pkgSet map[*ssa.Package]struct{}) *bytes.Buffer {
	graph := graphviz.New()

	// a, _ := graph.Graph()
	// foo, _ := a.CreateNode("Foo")
	// b, _ := graph.Graph()
	// bar, _ := b.CreateNode("Bar")
	// b.CreateEdge("e", foo, bar)
	// b.Close()
	// a.Close()

	g, err := graph.Graph()
	if err != nil {
		log.Fatalf("can't create a subgraph: %v", err)
	}
	defer func() {
		if err := g.Close(); err != nil {
			log.Fatalf("can't close the subgraph: %v", err)
		}
		graph.Close()
	}()

	log.Println("traversing the callgraph...")
	// callgraph.GraphVisitEdges()
	err = traverseExported(cgraph, pkgSet, func(e *callgraph.Edge) error {
		// child
		if e.Callee.String() == e.Caller.String() {
			return nil
		}
		if e.Callee.Func.Pkg == e.Caller.Func.Pkg {
			return nil
		}
		child, err := g.CreateNode(e.Caller.Func.Name())
		if err != nil {
			log.Fatal("can't create child node: %w", err)
		}
		// if _, ok := pkgSet[e.Caller.Func.Pkg]; ok {
		// 	child.SetLabel(e.Caller.Func.Name())
		// }

		// parent
		parent, err := g.CreateNode(e.Callee.Func.Name())
		if err != nil {
			log.Fatal("can't create parent node: %w", err)
		}
		// if _, ok := pkgSet[e.Callee.Func.Pkg]; ok {
		// 	parent.SetLabel(e.Callee.Func.Name())
		// }

		// edge
		_, err = g.CreateEdge("e", child, parent)
		if err != nil {
			log.Fatal("can't create edge: %w", err)
		}

		return nil
	})
	if err != nil {
		log.Fatalf("error traversing the callgraph: %v", err)
	}

	var buf bytes.Buffer
	if err := graph.Render(g, "dot", &buf); err != nil {
		log.Fatalf("error rendering Graphviz: %v", err)
	}

	return &buf
}
