# go get github.com/robertkrimen/godocdown/godocdown

NOVENDOR = $(shell go list ./... | grep -v vendor)

all: codegen

README.md:
	godocdown . > README.md
	echo "" >> README.md
	godocdown ./doctransport >> README.md
	echo "" >> README.md
	godocdown ./echotransport >> README.md

cleancodegen:
	rm -f README.md
.PHONY: clean

codegen: cleancodegen
	go fix $(NOVENDOR)
.PHONY: codegen

metalinter:
	gometalinter --config .gometalinter.json $(NOVENDOR)
.PHONY: metalinter

test:
	go test -v -cover $(NOVENDOR)
.PHONY: test

ci: codegen metalinter test
.PHONY: ci