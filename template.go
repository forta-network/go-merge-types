package merge

const codeTemplate = `
// Code generated by go-merge-types. DO NOT EDIT.

package {{.Output.Package}}

import (
	import_fmt "fmt"
	import_sync "sync"

{{range $source := .Sources}}
	{{$source.Package.Alias}} "{{$source.Package.ImportPath}}"
{{end}}

{{range $imp := .Output.Imports}}
	{{$imp}}
{{end}}
)

// {{.Output.Type}} is a new type which can multiplex calls to different implementation types.
type {{.Output.Type}} struct {
{{range $index, $source := .Sources}}
	typ{{$index}} *{{$source.Package.Alias}}.{{$source.Type}}
{{end}}
	currTag string
	mu import_sync.RWMutex
	unsafe bool // default: false
}

// New{{.Output.Type}} creates a new merged type.
func New{{.Output.Type}}({{range $index, $arg := .Output.InitArgs}}{{if eq $index 0}}{{else}}, {{end}}{{$arg.Name}} {{$arg.Type}}{{end}}) (*{{.Output.Type}}, error) {
	var (
		mergedType {{.Output.Type}}
		err error
	)
	mergedType.currTag = "{{.Output.DefaultTag}}"

{{range $sourceIndex, $source := .Sources}}
	mergedType.typ{{$sourceIndex}}, err = {{$source.Package.Alias}}.New{{$source.Type}}({{range $argIndex, $arg := $source.InitArgs}}{{if eq $argIndex 0}}{{else}}, {{end}}{{$arg.Name}}{{end}})
	if err != nil {
		return nil, import_fmt.Errorf("failed to initialize {{$source.Package.Alias}}.{{$source.Type}}: %v", err)
	}
{{end}}

	return &mergedType, nil
}

// Use sets the used implementation to given tag.
func (merged *{{.Output.Type}}) Use(tag string) (changed bool) {
	if !merged.unsafe {
		merged.mu.Lock()
		defer merged.mu.Unlock()
	}
	changed = merged.currTag != tag
	merged.currTag = tag
	return
}

// Unsafe disables the mutex.
func (merged *{{.Output.Type}}) Unsafe() {
	merged.unsafe = true
}

// Safe enables the mutex.
func (merged *{{.Output.Type}}) Safe() {
	merged.unsafe = false
}

{{range $method := .Output.Methods}}
{{if or $method.NoReturn $method.SingleReturn}}{{else}}
// {{$method.ReturnType.Name}} is a merged return type.
type {{$method.ReturnType.Name}} struct {
{{range $retField := $method.ReturnType.Fields}}
	{{$retField.Name}} {{$retField.Type}}
{{end}}
}{{end}}

// {{$method.Name}} multiplexes to different implementations of the method.
func (merged *{{$.Output.Type}}) {{$method.Name}}({{range $index, $arg := $method.Args}}{{if eq $index 0}}{{else}}, {{end}}{{$arg.Name}} {{$arg.Type}}{{end}}) {{if $method.NoReturn}}(err error){{else}}(retVal {{if eq $method.SingleReturn false}}*{{end}}{{$method.ReturnType.Name}}, err error){{end}} {
	if !merged.unsafe {
		merged.mu.RLock()
		defer merged.mu.RUnlock()
	}

{{if eq $method.SingleReturn false}}{{if eq $method.NoReturn false}}
	retVal = &{{$method.ReturnType.Name}}{}
{{end}}{{end}}

{{range $variation := $method.Variations}}
	if merged.currTag == "{{$variation.Tag}}" {
	{{if $variation.NoReturn}}{{else}}{{if $variation.OnlyError}}methodErr := {{else}}val, methodErr := {{end}}{{end}}merged.typ{{$variation.SourceIndex}}.{{$variation.Name}}({{range $index, $arg := $variation.Args}}{{if eq $index 0}}{{else}}, {{end}}{{$arg.Name}}{{end}})
{{if eq $variation.NoReturn false}}
		if err != nil {
			err = methodErr
			return
		}{{end}}
{{if $method.SingleReturn}}
		retVal = val
{{else}}
{{range $retField := $variation.ReturnedFields}}
		retVal.{{$retField.Name}} = val{{if $variation.MergeReturnedStruct}}.{{$retField.Name}}{{else}}{{end}}
{{end}}
{{end}}
		return
	}
{{end}}

	err = import_fmt.Errorf("{{$.Output.Type}}.{{$method.Name}} not implemented (tag=%s)", merged.currTag)
	return
}
{{end}}
`
