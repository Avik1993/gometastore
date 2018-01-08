## hmsbench doc

Generate documentation

### Synopsis

Generate documentation

```
hmsbench doc [flags]
```

### Options

```
      --dir string   doc directory (default "./doc")
  -h, --help         help for doc
```

### Options inherited from parent commands

```
      --config string     config file (default is $HOME/.hmsbench.yaml)
  -d, --database string   owner name
  -F, --filter string     run benchmarks matching the filter
  -H, --host string       hostname for HMS server (default "localhost")
  -N, --objects int       number of objects to create (default 100)
  -o, --output string     output file
  -P, --port string       port for HMS server (default "9083")
      --savedata string   location for raw benchmark data
  -u, --user string       owner name (default "user")
```

### SEE ALSO

* [hmsbench](hmsbench.md)	 - HMS Benchmarks

###### Auto generated by spf13/cobra on 5-Jan-2018