// Command rollops-plugin-cloudwatch is a Rollops metric-provider plugin backed
// by AWS CloudWatch. Build it, pin its sha256, and point a rollout's
// analysis.plugin at the binary.
package main

import (
	"fmt"
	"os"

	cloudwatch "github.com/klarlabs-studio/rollops-plugin-cloudwatch"
	"go.klarlabs.de/rollops/pkg/plugin"
)

// version is overwritten at build time via -ldflags.
var version = "dev"

func main() {
	safety := plugin.Safety{
		// Drives AWS via the aws CLI (ambient credentials / IAM role).
		EnvVars:   []string{"CLOUDWATCH_AWS", "CLOUDWATCH_REGION", "CLOUDWATCH_WINDOW", "AWS_REGION", "AWS_PROFILE"},
		RiskClass: plugin.RiskPassive, // reads metrics only
	}
	if err := plugin.ServeMetricProvider("klarlabs/cloudwatch", version, cloudwatch.FromEnv(), safety); err != nil {
		fmt.Fprintln(os.Stderr, "rollops-plugin-cloudwatch:", err)
		os.Exit(1)
	}
}
