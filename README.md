# decl
Declarative command-line and environment processing in Go

Instead of manually checking and parsing env vars and command line switches, with `decl`, you just define everything you're expecting (along with their default values) and the library does it all for you

```go
type MyArgs struct {
    Verbose bool     `arg:"v,verbose"`
    Files   []string `arg:"*"`
}

type MyEnv struct {
    AwsApiKey string `env:"AWS_API_KEY"`   // required, as no default value provided
    AwsSecret string `env:"AWS_SECRET"`    // ditto
    Shell     string `env:"SHELL=/bin/sh"` // optional, as a default value is provided
    Explode   bool   `env:"EXPLODE=0"`     // ditto
}

func main() {
    args := MyArgs{}
    LoadArgsSpec(&args)

    env := MyEnv{}
    LoadEnvSpec(&env)
    
    ...
}
```

![Go'al nefesh](/goalnefesh.jpg "Go'al nefesh").

