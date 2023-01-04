package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/forta-network/go-merge-types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var mainCmd = &cobra.Command{
	Use:          "gomergetypes",
	Long:         `Generate code that merges the interfaces and multiplexes to different implementations`,
	Run:          handleMain,
	SilenceUsage: true,
}

var (
	flagConfigPath *string
	flagVerbose    *bool
)

func handleMain(cmd *cobra.Command, args []string) {
	b, err := os.ReadFile(*flagConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	var config merge.MergeConfig
	if err := yaml.Unmarshal(b, &config); err != nil {
		log.Fatal(err)
	}

	// fix package source dirs relative to the config path
	for _, source := range config.Sources {
		source.Package.SourceDir = relativePath(source.Package.SourceDir)
	}

	b, err = merge.Generate(&config)
	if err != nil {
		log.Fatal(err)
	}

	if *flagVerbose {
		fmt.Println(string(b))
	}

	if err := ioutil.WriteFile(relativePath(config.Output.File), b, 0755); err != nil {
		log.Fatal(err)
	}
}

func main() {
	flagConfigPath = mainCmd.Flags().String("config", "gomergetypes.yml", "config file path")
	flagVerbose = mainCmd.Flags().BoolP("verbose", "v", false, "verbose output")
	if err := mainCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func relativePath(input string) string {
	return path.Join(path.Dir(*flagConfigPath), input)
}
