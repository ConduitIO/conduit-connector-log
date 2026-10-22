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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
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
	// this connector). When disabled, only a salted hash of the position and
	// of the key, the operation and an allowlist of safe metadata keys are
	// logged; every other metadata value is dropped.
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
		// Default: never write key, payload or free-form metadata values
		// (GHSA-5c9c-5rf5-2r5q). Only identifiers that are known to be safe are
		// logged as they are; everything else is dropped or hashed.
		logger.WithLevel(d.level).
			Str("position_hash", saltedHash(r.Position)).
			Str("operation", r.Operation.String()).
			Str("key_hash", keyHash(r)).
			Dict("metadata", safeMetadata(r.Metadata)).
			Msg(d.config.Message)
	}
	return len(records), nil
}

// salt is random per process. Hashes are stable within one run, so records can
// be correlated, and differ across runs, so a hash of a low-entropy key or
// position cannot be brute-forced offline from logs alone.
var salt = func() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("log connector: cannot read random bytes: %w", err))
	}
	return b
}()

// saltedHash returns the hex HMAC-SHA-256 of data under the process salt.
func saltedHash(data []byte) string {
	m := hmac.New(sha256.New, salt)
	m.Write(data)
	return hex.EncodeToString(m.Sum(nil))
}

// keyHash returns the salted hash of the record key, or "" if there is no key.
func keyHash(r opencdc.Record) string {
	if r.Key == nil {
		return ""
	}
	return saltedHash(r.Key.Bytes())
}

// safeMetadataKeys are the metadata keys whose values are logged as they are:
// timestamps, versions, names and ids that Conduit and the connectors set and
// that do not carry record data. Every other key's value is dropped.
var safeMetadataKeys = map[string]struct{}{
	opencdc.MetadataOpenCDCVersion:                  {},
	opencdc.MetadataCreatedAt:                       {},
	opencdc.MetadataReadAt:                          {},
	opencdc.MetadataCollection:                      {},
	opencdc.MetadataConduitSourcePluginName:         {},
	opencdc.MetadataConduitSourcePluginVersion:      {},
	opencdc.MetadataConduitDestinationPluginName:    {},
	opencdc.MetadataConduitDestinationPluginVersion: {},
	opencdc.MetadataConduitSourceConnectorID:        {},
	opencdc.MetadataConduitDLQNackNodeID:            {},
}

// goQualifier matches the start of a Go error such as "strconv.Atoi: parsing
// ...": a lower-case package, a dot and an exported identifier.
var goQualifier = regexp.MustCompile(`^[a-z][a-z0-9]*\.[A-Z][A-Za-z0-9]*$`)

// safeMetadata returns the metadata that may be logged without payload: the
// allowlisted keys with their values, a salted hash and the Go qualifier (never
// the text) of the DLQ nack error, and the number of entries that were dropped.
// The text of a nack error routinely echoes the value that failed to parse.
func safeMetadata(md map[string]string) *zerolog.Event {
	keys := make([]string, 0, len(md))
	for k := range md {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ev := zerolog.Dict()
	dropped := 0
	for _, k := range keys {
		v := md[k]
		switch k {
		case opencdc.MetadataConduitDLQNackError:
			// Go type or code only, never the text.
			kind := "redacted"
			if head, _, ok := strings.Cut(v, ": "); ok && goQualifier.MatchString(head) {
				kind = head
			}
			ev.Str(k, kind).Str(k+".hash", saltedHash([]byte(v)))
		default:
			if _, ok := safeMetadataKeys[k]; ok {
				ev.Str(k, v)
			} else {
				dropped++
			}
		}
	}
	if dropped > 0 {
		ev.Int("dropped", dropped)
	}
	return ev
}

func (d *Destination) Teardown(_ context.Context) error {
	return nil // nothing to tear down
}
