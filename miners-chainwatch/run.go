package main

import (
	"encoding/json"
	"fmt"
	"github.com/phantom-rabbit/filecoin/miners-chainwatch/syncer"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"

	"github.com/phantom-rabbit/filecoin/miners-chainwatch/models"
	"github.com/phantom-rabbit/filecoin/util"
)

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start lotus chainwatch",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "max-batch",
			Value: 50,
		},
	},
	Action: func(cctx *cli.Context) error {
		go func() {
			http.ListenAndServe(":6060", nil) //nolint:errcheck
		}()
		ll := cctx.String("log-level")
		if err := logging.SetLogLevel("*", ll); err != nil {
			return err
		}
		if err := logging.SetLogLevel("rpc", "error"); err != nil {
			return err
		}

		var api api.FullNode
		var closer jsonrpc.ClientCloser
		var err error

		if tokenMaddr := cctx.String("api"); tokenMaddr != "" {
			toks := strings.Split(tokenMaddr, ":")

			api, closer, err = util.GetFullNodeAPIUsingCredentials(cctx.Context, toks[1], toks[0])
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("invalid api tokens, expected <token>:<maddr>, got: %s", tokenMaddr)
		}
		defer closer()

		v, err := api.Version(cctx.Context)
		if err != nil {
			return err
		}

		log.Infof("Remote version: %s", v.Version)

		var config models.MgoConfig
		if dbInfo := cctx.String("db"); dbInfo != "" {
			err := json.Unmarshal([]byte(dbInfo), &config)
			if err != nil {
				return fmt.Errorf("parse db config fall:%s", err)
			}
		} else {
			indent, _ := json.MarshalIndent(config, " ", " ")
			return fmt.Errorf("invalid db config, expected \n %s \n but got: %s", indent, dbInfo)
		}

		if err := models.Init(config); err != nil {
			return err
		}

		var miners []string
		if m := cctx.String("miners"); m != "" {
			miners = strings.Split(strings.TrimSpace(m), ",")
		}

		syncer, err := syncer.NewSyncer(miners, cctx.Int("max-batch"), cctx.Int("db"), api)
		if err != nil {
			return err
		}

		go syncer.Start(cctx.Context)

		<-cctx.Done()
		os.Exit(0)
		return nil
	},
}