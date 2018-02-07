package main

import (
	"github.com/coreos/operator-sdk/pkg/generator"
)

func main() {
	g := &generator.Generator{}
	err := g.Render()
	if err != nil {
		panic(err)
	}
}
