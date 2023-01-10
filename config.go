package merge

import (
	"github.com/forta-network/go-merge-types/rewrite"
)

type MergeConfig struct {
	Sources []*Source `yaml:"sources"`
	Output  Output    `yaml:"output"`
}

type Source struct {
	Type     string   `yaml:"type"`
	Tag      string   `yaml:"tag"`
	Package  Package  `yaml:"package"`
	InitArgs []*Field `yaml:"-"`
}

type Package struct {
	ImportPath string `yaml:"importPath"`
	Alias      string `yaml:"alias"`
	SourceDir  string `yaml:"sourceDir"`
}

type Output struct {
	Type       string           `yaml:"type"`
	Package    string           `yaml:"package"`
	File       string           `yaml:"file"`
	Rewrite    rewrite.Rewriter `yaml:"rewrite"`
	DefaultTag string           `yaml:"defaultTag"`

	InitArgs []*Field  `yaml:"-"`
	Methods  []*Method `yaml:"-"`
	Imports  []string  `yaml:"-"`
}

type Field struct {
	SourceIndex int
	Name        string
	Type        string
}

type ReturnType struct {
	Name   string
	Fields []*Field
}

type Method struct {
	Name         string
	Variations   []*Variation
	Args         []*Field
	ReturnType   ReturnType
	NoReturn     bool
	SingleReturn bool
}

type Variation struct {
	Name                string
	SourceIndex         int
	Tag                 string
	Args                []*Field
	ReturnedFields      []*Field
	MergeReturnedStruct bool
	NoReturn            bool
	OnlyError           bool
}
