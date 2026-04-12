package main

import (
	"fmt"

	dndengine "github.com/zwh8800/dnd-core/pkg/engine"
)

func main() {
	e, err := dndengine.New(dndengine.DefaultConfig())
	if err != nil {
		panic(err)
	}
	fmt.Println(e)
}
