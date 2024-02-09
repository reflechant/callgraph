# callgraph

A work-in-progress tool to generate readable function callgraphs for Golang packages. It's not intended to be comprehensive, only to show dependencies between packages at exported functions level. It allows to assess architecture complexity and detect dependency antipatterns.

## Usage

```
go run main.go -d <package_path> | dot -Tsvg -o graph.svg
```
