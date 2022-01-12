package main

//go:generate go-bindata -o migration/bindata.go -nometadata -pkg migration -ignore bindata.go migration/
//go:generate go fmt ./migration/bindata.go
//go:generate goimports -l ./migration/bindata.go

import (
	_ "github.com/jteeuwen/go-bindata" // so it's detected by `dep ensure`
	"github.com/lbryio/chainquery/cmd"
)

func main() {
	cmd.Execute()
}
