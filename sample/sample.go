package main

import (
	"fmt"

	declargs "github.com/tomerfiliba/decl/args"
	declenv "github.com/tomerfiliba/decl/env"
)

type MyArgs struct {
	Verbose bool `arg:"v,verbose"`
}

type MyEnv struct {
	User  string `env:"USER"`
	Shell string `env:"SHELL=/bin/sh"`
}

// /////////////////////////////////////////////////////////////////////////////
//
// $ go run sample/sample.go
// verbose=false SHELL=/usr/bin/zsh USER=tomer
//
// $ go run sample/sample.go -v
// verbose=true SHELL=/usr/bin/zsh USER=tomer
//
// /////////////////////////////////////////////////////////////////////////////
func main() {
	args := MyArgs{}
	declargs.LoadArgsSpec(&args)

	env := MyEnv{}
	declenv.LoadEnvSpec(&env)

	fmt.Printf("verbose=%v SHELL=%s USER=%s\n", args.Verbose, env.Shell, env.User)
}
