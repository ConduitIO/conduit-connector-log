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

package log

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/rs/zerolog"
)

const (
	ConfigLevel   = "level"
	LevelDefault  = zerolog.InfoLevel
	ConfigMessage = "message"
)

type Destination struct {
	sdk.UnimplementedDestination

	level zerolog.Level
	msg   string
}

func NewDestination() sdk.Destination {
	return &Destination{}
}

func (d *Destination) Parameters() map[string]sdk.Parameter {
	return map[string]sdk.Parameter{
		ConfigLevel: {
			Default:     "INFO",
			Required:    false,
			Description: "The log level used to log records.",
		},
		ConfigMessage: {
			Required:    false,
			Description: "Optional message that should be added to the log output of every record.",
		},
	}
}

func (d *Destination) Configure(ctx context.Context, cfg map[string]string) error {
	level := LevelDefault
	if l := cfg[ConfigLevel]; l != "" {
		var err error
		level, err = zerolog.ParseLevel(strings.ToLower(l))
		if err != nil {
			return fmt.Errorf(
				"%q config value %q does not contain a valid level: %w",
				ConfigLevel, l, err,
			)
		}
	}
	d.level = level
	d.msg = cfg[ConfigMessage]
	return nil
}

func (d *Destination) Open(ctx context.Context) error {
	return nil // nothing to open
}

func (d *Destination) Write(ctx context.Context, records []sdk.Record) (int, error) {
	logger := sdk.Logger(ctx)
	for _, r := range records {
		logger.WithLevel(d.level).
			RawJSON("record", r.Bytes()).
			Msg(d.msg)
	}
	return len(records), nil
}

func (d *Destination) Teardown(ctx context.Context) error {
	return nil // nothing to tear down
}
