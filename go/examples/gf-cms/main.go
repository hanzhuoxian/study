package main

import (
	_ "gf-cms/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"gf-cms/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
