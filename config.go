package cloudwatch

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// FromEnv builds a Provider that drives AWS via the aws CLI using ambient
// credentials (env, profile, or IAM role) — the Rollops target spec carries only
// the JSON metric query, never AWS credentials.
//
//	CLOUDWATCH_AWS     aws binary to use (default "aws")
//	CLOUDWATCH_REGION  optional --region override (else the CLI's ambient region)
//	CLOUDWATCH_WINDOW  lookback window as a Go duration (default 5m)
func FromEnv() Provider {
	win, _ := time.ParseDuration(os.Getenv("CLOUDWATCH_WINDOW"))
	return Provider{
		AWS:    os.Getenv("CLOUDWATCH_AWS"),
		Region: os.Getenv("CLOUDWATCH_REGION"),
		Window: win,
		Run:    execRunner,
	}
}

func execRunner(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, args[0], args[1:]...).Output()
	return string(out), err
}
