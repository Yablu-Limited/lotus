package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/api"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/tablewriter"
)

var dagstoreCmd = &cli.Command{
	Name:  "dagstore",
	Usage: "Manage the dagstore on the markets subsystem",
	Subcommands: []*cli.Command{
		dagstoreListShardsCmd,
		dagstoreInitializeShardCmd,
		dagstoreInitializeAllCmd,
		dagstoreGcCmd,
	},
}

var dagstoreListShardsCmd = &cli.Command{
	Name:  "list-shards",
	Usage: "List all shards known to the dagstore, with their current status",
	Action: func(cctx *cli.Context) error {
		marketsApi, closer, err := lcli.GetMarketsAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := lcli.ReqContext(cctx)

		shards, err := marketsApi.DagstoreListShards(ctx)
		if err != nil {
			return err
		}

		if len(shards) == 0 {
			return nil
		}

		tw := tablewriter.New(
			tablewriter.Col("Key"),
			tablewriter.Col("State"),
			tablewriter.Col("Error"),
		)

		colors := map[string]color.Attribute{
			"ShardStateAvailable": color.FgGreen,
			"ShardStateServing":   color.FgBlue,
			"ShardStateErrored":   color.FgRed,
			"ShardStateNew":       color.FgYellow,
		}

		for _, s := range shards {
			m := map[string]interface{}{
				"Key": s.Key,
				"State": func() string {
					if c, ok := colors[s.State]; ok {
						return color.New(c).Sprint(s)
					}
					return s.State
				}(),
				"Error": s.Error,
			}
			tw.Write(m)
		}

		return tw.Flush(os.Stdout)
	},
}

var dagstoreInitializeShardCmd = &cli.Command{
	Name:      "initialize-shard",
	ArgsUsage: "[key]",
	Usage:     "Initialize the specified shard",
	Action: func(cctx *cli.Context) error {
		if cctx.NArg() != 1 {
			return fmt.Errorf("must provide a single shard key")
		}

		marketsApi, closer, err := lcli.GetMarketsAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := lcli.ReqContext(cctx)

		return marketsApi.DagstoreInitializeShard(ctx, cctx.Args().First())
	},
}

var dagstoreInitializeAllCmd = &cli.Command{
	Name:  "initialize-all",
	Usage: "Initialize all uninitialized shards, streaming results as they're produced",
	Flags: []cli.Flag{
		&cli.UintFlag{
			Name:     "concurrency",
			Usage:    "maximum shards to initialize concurrently at a time; use 0 for unlimited",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		concurrency := cctx.Uint("concurrency")

		marketsApi, closer, err := lcli.GetMarketsAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := lcli.ReqContext(cctx)

		params := api.DagstoreInitializeAllParams{
			MaxConcurrency: int(concurrency),
		}

		ch, err := marketsApi.DagstoreInitializeAll(ctx, params)
		if err != nil {
			return err
		}

		for {
			select {
			case res, ok := <-ch:
				if !ok {
					return nil
				}
				if res.Success {
					_, _ = fmt.Fprintln(os.Stdout, res.Key, color.New(color.FgGreen).Sprint("SUCCESS"))
				} else {
					_, _ = fmt.Fprintln(os.Stdout, res.Key, color.New(color.FgRed).Sprint("ERROR"), res.Error)
				}

			case <-ctx.Done():
				return fmt.Errorf("aborted")
			}
		}
	},
}

var dagstoreGcCmd = &cli.Command{
	Name:  "gc",
	Usage: "Garbage collect the dagstore",

	Action: func(cctx *cli.Context) error {
		marketsApi, closer, err := lcli.GetMarketsAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := lcli.ReqContext(cctx)

		collected, err := marketsApi.DagstoreGC(ctx)
		if err != nil {
			return err
		}

		if len(collected) == 0 {
			_, _ = fmt.Fprintln(os.Stdout, "no shards collected")
			return nil
		}

		for _, e := range collected {
			if e.Error == "" {
				_, _ = fmt.Fprintln(os.Stdout, e.Key, "success")
			} else {
				_, _ = fmt.Fprintln(os.Stdout, e.Key, "failed:", e.Error)
			}
		}

		return nil
	},
}
