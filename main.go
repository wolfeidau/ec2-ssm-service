package main

import (
	"context"
	"os"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/ec2-ssm-service/internal/ssmfile"
	"gopkg.in/yaml.v3"
)

var cli struct {
	Version kong.VersionFlag
	DryRun  bool `help:"Dry run, do not write any files"`
	// an array of key value pairs containing an SSM key and a path to a file
	Config struct {
		ConfigFile map[string]string `arg:"" type:":file" help:"SSM key and configuration path pairs"`
	} `cmd:"" help:"Write configuration files from SSM parameters"`
	Env struct {
		EnvFile map[string]string `arg:"" type:":file" help:"Environment file path"`
	} `cmd:"" help:"Write environment files from SSM parameters"`
}

func main() {
	ctx := context.Background()
	cliCtx := kong.Parse(&cli,
		kong.Configuration(
			kong.JSON,
			"/etc/ec2-ssm-config-service.yaml",
			"~/.ec2-ssm-config-service.yaml",
		),
	)

	awscfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load aws config")
	}

	// new aws sdkv2 client
	ssmsvc := ssm.NewFromConfig(awscfg)

	// new ssmfile batcher
	bt := ssmfile.NewBatcher(ssmsvc)

	if cli.DryRun {
		log.Info().Msg("dry run enabled, not writing any files")

		err = yaml.NewEncoder(os.Stdout).Encode(&cli)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to encode cli struct")
		}

		return
	}

	switch cliCtx.Command() {
	case "config <config-file>":

		// write the configs to the files
		if err := bt.WriteConfigs(ctx, cli.Config.ConfigFile); err != nil {
			log.Fatal().Err(err).Msg("failed to write config files")
		}
	case "env <env-file>":
		// write the environment variables to the files
		if err := bt.WriteEnvFiles(ctx, cli.Env.EnvFile); err != nil {
			log.Fatal().Err(err).Msg("failed to write environment files")
		}
	}

}
