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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/rs/zerolog"
)

type Destination struct {
	sdk.UnimplementedDestination

	config DestinationConfig
	level  zerolog.Level
}

func NewDestination() sdk.Destination {
	return sdk.DestinationWithMiddleware(&Destination{})
}

type DestinationConfig struct {
	sdk.DefaultDestinationMiddleware

	// The log level used to log records.
	Level string `json:"level" default:"info" validate:"inclusion=trace|debug|info|warn|error"`
	// Optional message that should be added to the log output of every record.
	Message string `json:"message"`
	// Log the full record, including key and payload. Disabled by default
	// because log output is commonly shipped to systems that are not
	// authorized to read record data (the default Conduit DLQ writes through
	// this connector). When disabled, only the position, operation, a SHA-256
	// hash of the key and the metadata are logged.
	Payload bool `json:"payload" default:"false"`
}

func (c DestinationConfig) Validate(ctx context.Context) error {
	var errs []error

	_, err := c.LogLevel()
	if err != nil {
		errs = append(errs, err)
	}

	errs = append(errs, c.DefaultDestinationMiddleware.Validate(ctx))
	return errors.Join(errs...)
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

func (d *Destination) Config() sdk.DestinationConfig {
	return &d.config
}

func (d *Destination) Open(_ context.Context) error {
	d.level, _ = d.config.LogLevel()
	return nil
}

func (d *Destination) Write(ctx context.Context, records []opencdc.Record) (int, error) {
	logger := sdk.Logger(ctx)
	for _, r := range records {
		if d.config.Payload {
			logger.WithLevel(d.level).
				RawJSON("record", r.Bytes()).
				Msg(d.config.Message)
			continue
		}
		// Default: never write key or payload data (GHSA-5c9c-5rf5-2r5q).
		logger.WithLevel(d.level).
			Bytes("position", r.Position).
			Str("operation", r.Operation.String()).
			Str("key_sha256", keyHash(r)).
			Interface("metadata", r.Metadata).
			Msg(d.config.Message)
	}
	return len(records), nil
}

// keyHash returns the hex SHA-256 of the record key, or "" if there is no key.
// The hash lets operators correlate records without exposing key contents.
// Note that hashes of low-entropy keys can be brute-forced.
func keyHash(r opencdc.Record) string {
	if r.Key == nil {
		return ""
	}
	sum := sha256.Sum256(r.Key.Bytes())
	return hex.EncodeToString(sum[:])
}

func (d *Destination) Teardown(_ context.Context) error {
	return nil // nothing to tear down
}
