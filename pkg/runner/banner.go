package runner

import "github.com/projectdiscovery/gologger"

const (
	banner = `
               __   __    ____               
   ___  ___ _ / /_ / /   / __/____ ___ _ ___ 
  / _ \/ _  // __// _ \ _\ \ / __// _  // _ \
 / .__/\_,_/ \__//_//_//___/ \__/ \_,_//_//_/
/_/
`
	Version          = `1.0.1`
	userName         = "wjlin0"
	pathScanRepoName = "deadpool"
	toolName         = "deadpool"
)

func showBanner() {
	gologger.Print().Msgf("%s\n", banner)
	gologger.Print().Msgf("\t\t\twjlin0.com\n\n")
	gologger.Print().Msgf("慎用。你要为自己的行为负责\n")
	gologger.Print().Msgf("开发者不承担任何责任，也不对任何误用或损坏负责.\n")
}
