package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"github.com/forta-network/go-merge-types"
	"github.com/forta-network/go-merge-types/utils"
	"github.com/spf13/cobra"
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
	config, b, err := merge.Run(*flagConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	if *flagVerbose {
		fmt.Println(string(b))
	}

	if err := ioutil.WriteFile(utils.RelativePath(*flagConfigPath, config.Output.File), b, 0755); err != nil {
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
