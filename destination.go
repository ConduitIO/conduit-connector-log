// Copyright © 2022 Meroxa, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate paramgen -output destination_paramgen.go DestinationConfig

package log

import (
	"context"
	"fmt"
	"strings"

	"github.com/conduitio/conduit-commons/opencdc"

	"github.com/conduitio/conduit-commons/config"

	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/rs/zerolog"
)

type Destination struct {
	sdk.UnimplementedDestination

	config DestinationConfig
	level  zerolog.Level
}

func NewDestination() sdk.Destination {
	return sdk.DestinationWithMiddleware(&Destination{}, sdk.DefaultDestinationMiddleware()...)
}

type DestinationConfig struct {
	// The log level used to log records.
	Level string `json:"level" default:"info" validate:"inclusion=trace|debug|info|warn|error"`
	// Optional message that should be added to the log output of every record.
	Message string `json:"message"`
}

func (c DestinationConfig) LogLevel() (zerolog.Level, error) {
	level, err := zerolog.ParseLevel(strings.ToLower(c.Level))
	if err != nil {
		return 0, fmt.Errorf(
			"%q config value %q does not contain a valid level: %w",
			"level", c.Level, err,
		)
	}
	return level, nil
}

func (d *Destination) Parameters() config.Parameters {
	return d.config.Parameters()
}

func (d *Destination) Configure(ctx context.Context, cfg config.Config) error {
	err := sdk.Util.ParseConfig(ctx, cfg, &d.config, NewDestination().Parameters())
	if err != nil {
		return err
	}
	d.level, err = d.config.LogLevel()
	if err != nil {
		return err
	}
	return nil
}

func (d *Destination) Open(_ context.Context) error {
	return nil // nothing to open
}

func (d *Destination) Write(ctx context.Context, records []opencdc.Record) (int, error) {
	logger := sdk.Logger(ctx)
	for _, r := range records {
		logger.WithLevel(d.level).
			RawJSON("record", r.Bytes()).
			Msg(d.config.Message)
	}
	return len(records), nil
}

func (d *Destination) Teardown(_ context.Context) error {
	return nil // nothing to tear down
}
