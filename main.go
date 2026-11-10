package main

import (
	"embed"

	"github.com/yanghui1-arch/lychee/internal"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := app.Run(assets); err != nil {
		println("Error:", err.Error())
	}
}
