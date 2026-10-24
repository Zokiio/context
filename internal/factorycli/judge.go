package factorycli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/Zokiio/context/internal/factoryworker"
	cli "github.com/urfave/cli/v3"
)

func judgeFlags() []cli.Flag {
	return []cli.Flag{&cli.StringFlag{Name: "judge-command-file", Usage: "JSON argv of an independent read-only dispute judge"}}
}

func soloJudge(c *cli.Command) ([]string, error) {
	if c.Bool("no-judge") {
		return nil, nil
	}
	if argv, err := commandFile(c.String("judge-command-file")); err != nil || argv != nil {
		return argv, err
	}
	if _, err := exec.LookPath("claude"); err != nil {
		return nil, nil
	}
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return []string{self, "factory", "judge", "--model", c.String("judge-model"), "--max-budget-usd", fmt.Sprint(c.Float64("judge-budget-usd"))}, nil
}

func judgeCommand(stdout, stderr io.Writer) *cli.Command {
	return &cli.Command{Name: "judge", Hidden: true, Usage: "Judge the captured dispute on stdin with independent read-only Claude Code",
		Flags: []cli.Flag{&cli.StringFlag{Name: "binary"}, &cli.StringFlag{Name: "model"}, &cli.Float64Flag{Name: "max-budget-usd"}},
		Action: func(ctx context.Context, c *cli.Command) error {
			return factoryworker.Judge(ctx, os.Stdin, stdout, stderr, factoryworker.JudgeOptions{Binary: c.String("binary"), Model: c.String("model"), MaxBudgetUSD: c.Float64("max-budget-usd")})
		}}
}
