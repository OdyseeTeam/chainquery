package main

//go:generate go-bindata -nometadata -o migration/bindata.go -pkg migration -ignore bindata.go migration/
//go:generate go fmt ./migration/bindata.go
//go:generate goimports -l ./migration/bindata.go

import (
	"github.com/lbryio/chainquery/cmd"
)

func main() {
	cmd.Execute()
}
