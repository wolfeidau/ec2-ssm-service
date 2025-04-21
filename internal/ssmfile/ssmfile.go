package ssmfile

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/rs/zerolog/log"
)

const (
	envFilePerm    = 0644
	configFilePerm = 0644
)

type SSMClient interface {
	GetParameters(ctx context.Context, params *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error)
	GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}

type Batcher struct {
	ssmsvc SSMClient // aws sdk v2 ssm client
}

func NewBatcher(ssmsvc SSMClient) *Batcher {
	return &Batcher{
		ssmsvc: ssmsvc,
	}
}

func (bt *Batcher) WriteConfigs(ctx context.Context, files map[string]string) error {
	names := slices.Sorted(maps.Keys(files))

	// build a list of ssm keys to retrieve
	getRes, err := bt.ssmsvc.GetParameters(ctx, &ssm.GetParametersInput{
		Names:          names,
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to get parameters: %w", err)
	}

	// write the values to the files
	for _, param := range getRes.Parameters {
		log.Info().Str("name", aws.ToString(param.Name)).Int64("version", param.Version).Msg("writing config")

		if err := writeConfigFile(files[aws.ToString(param.Name)], aws.ToString(param.Value), aws.ToTime(param.LastModifiedDate)); err != nil {
			return fmt.Errorf("failed to write file %s: %w", aws.ToString(param.Name), err)
		}
	}

	return nil
}

// search for keys below the provided path (1 layer), trim off the prefix and create an env
// file containing the values
func (bt *Batcher) WriteEnvFiles(ctx context.Context, envFiles map[string]string) error {

	// for each env file
	for envPath, envFile := range envFiles {
		// get all keys below the path
		getRes, err := bt.ssmsvc.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:           aws.String(envPath),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true),
			MaxResults:     aws.Int32(10),
		})
		if err != nil {
			return fmt.Errorf("failed to get parameters by path: %w", err)
		}

		// for each value trim the path and build a list of envs to write to a file
		envs := make(map[string]string)
		for _, param := range getRes.Parameters {
			envName := strings.TrimPrefix(aws.ToString(param.Name), envPath)
			envs[envName] = aws.ToString(param.Value)
		}

		err = writeEnvFile(envFile, envs)
		if err != nil {
			return fmt.Errorf("failed to write env file %s: %w", envFile, err)
		}
	}

	return nil
}

func writeConfigFile(path, value string, lastModified time.Time) error {
	if err := os.WriteFile(path, []byte(value), configFilePerm); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	// update the files last modified
	if err := os.Chtimes(path, lastModified, lastModified); err != nil {
		return fmt.Errorf("failed to update file %s: %w", path, err)
	}
	return nil
}

func writeEnvFile(envFile string, envs map[string]string) error {
	// open the envfile
	f, err := os.OpenFile(envFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, envFilePerm)
	if err != nil {
		return fmt.Errorf("failed to open env file %s: %w", envFile, err)
	}
	defer func() { _ = f.Close() }()

	// write the envs to the file
	for envName, envVal := range envs {
		if _, err := f.WriteString(fmt.Sprintf("%s=\"%s\"\n", envName, envVal)); err != nil {
			return fmt.Errorf("failed to write env %s: %w", envName, err)
		}
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close env file %s: %w", envFile, err)
	}

	return nil
}
