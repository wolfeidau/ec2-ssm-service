package main

import (
	"context"
	"io"
	"os"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/ec2-ssm-service/internal/ssmfile"
	"gopkg.in/yaml.v3"
)

var (
	version = "dev"

	cli struct {
		Version      kong.VersionFlag
		DryRun       bool  `help:"Dry run, do not write any files"`
		Batch        int32 `flag:"batch" help:"Batch size for fetching SSM parameters" default:"10"`
		EC2Discovery bool  `name:"ec2-discovery" help:"Enable EC2 metadata discovery"`
		// an array of key value pairs containing an SSM key and a path to a file
		// the key is the SSM parameter name and the value is the path to the file
		// the path is relative to the root of the filesystem
		Configs map[string]string `flag:"config" help:"SSM key and configuration target path" yaml:"configs"`
		// an array of key value pairs containing an SSM key and a path to a file
		// the key is the SSM parameter name and the value is the path to the file
		// the path is relative to the root of the filesystem
		EnvFiles map[string]string `flag:"env-file" help:"SSM key prefix for environment variables and env file target path" yaml:"env-files"`
	}
)

func main() {
	ctx := context.Background()
	kong.Parse(&cli,
		kong.Vars{"version": version},
		kong.Configuration(
			kongyaml.Loader,
			"/etc/ec2-ssm-config-service.yaml",
			"~/.ec2-ssm-config-service.yaml",
		),
	)

	awscfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load aws config")
	}

	if cli.EC2Discovery {
		client := imds.NewFromConfig(awscfg)

		region, err := client.GetMetadata(context.TODO(), &imds.GetMetadataInput{
			Path: "placement/region",
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get AWS Region in which the instance is launched")
		}

		content, _ := io.ReadAll(region.Content)

		awscfg.Region = string(content)
	}

	// new aws sdkv2 client
	ssmsvc := ssm.NewFromConfig(awscfg)

	// new ssmfile batcher
	bt := ssmfile.NewBatcher(ssmsvc, cli.Batch)

	if cli.DryRun {
		log.Info().Msg("dry run enabled, not writing any files")

		err = yaml.NewEncoder(os.Stdout).Encode(&cli)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to encode cli struct")
		}

		return
	}

	// write the configs to the files
	if err := bt.WriteConfigs(ctx, cli.Configs); err != nil {
		log.Fatal().Err(err).Msg("failed to write config files")
	}

	// write the environment variables to the files
	if err := bt.WriteEnvFiles(ctx, cli.EnvFiles); err != nil {
		log.Fatal().Err(err).Msg("failed to write environment files")
	}

}
