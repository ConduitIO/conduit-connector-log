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
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
)

func TestDestination_Configure_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	testCases := map[string]zerolog.Level{
		"":      zerolog.InfoLevel,
		"error": zerolog.ErrorLevel,
		"warn":  zerolog.WarnLevel,
		"info":  zerolog.InfoLevel,
		"debug": zerolog.DebugLevel,
		"trace": zerolog.TraceLevel,
	}

	for have, want := range testCases {
		t.Run(have, func(_ *testing.T) {
			var dest Destination
			err := sdk.Util.ParseConfig(
				ctx,
				map[string]string{"level": have},
				dest.Config(),
				Connector.NewSpecification().DestinationParams,
			)
			is.NoErr(err)
			level, err := dest.config.LogLevel()
			is.NoErr(err)
			is.Equal(level, want)
		})
	}
}

func TestDestination_Configure_Fail(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	var dest Destination
	err := sdk.Util.ParseConfig(
		ctx,
		map[string]string{"level": "invalid"},
		dest.Config(),
		Connector.NewSpecification().DestinationParams,
	)
	is.True(err != nil)
}

func TestDestination_Configure_Write(t *testing.T) {
	is := is.New(t)

	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.InfoLevel) // ignore debug logs
	ctx := logger.WithContext(context.Background())      // sdk takes logger from the context

	var dest Destination

	err := sdk.Util.ParseConfig(
		ctx,
		map[string]string{"level": "warn", "message": "foo!", "payload": "true"},
		dest.Config(),
		Connector.NewSpecification().DestinationParams,
	)
	is.NoErr(err)

	have := []opencdc.Record{{
		Position:  []byte("r1"),
		Operation: opencdc.OperationCreate,
		Metadata:  map[string]string{"foo1": "bar1"},
		Key:       opencdc.RawData("raw-key"),
		Payload: opencdc.Change{
			Before: nil,
			After:  opencdc.RawData("raw-payload"),
		},
	}, {
		Position:  []byte("r2"),
		Operation: opencdc.OperationUpdate,
		Metadata:  map[string]string{"foo2": "bar2"},
		Key:       opencdc.StructuredData{"r2-key": 1},
		Payload: opencdc.Change{
			Before: opencdc.StructuredData{"r2-payload": true},
			After:  opencdc.StructuredData{"r2-payload": false},
		},
	}}

	want := []string{
		`{"level":"warn","record":{"position":"cjE=","operation":"create","metadata":{"foo1":"bar1"},"key":"cmF3LWtleQ==","payload":{"before":null,"after":"cmF3LXBheWxvYWQ="}},"message":"foo!"}`,
		`{"level":"warn","record":{"position":"cjI=","operation":"update","metadata":{"foo2":"bar2"},"key":{"r2-key":1},"payload":{"before":{"r2-payload":true},"after":{"r2-payload":false}}},"message":"foo!"}`,
	}

	l, err := dest.Write(ctx, have)
	is.NoErr(err)
	is.Equal(l, len(have)) // expected destination to write all records

	i := 0
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		is.Equal(scanner.Text(), want[i])
		i++
	}
}

func newTestRecords() []opencdc.Record {
	return []opencdc.Record{{
		Position:  []byte("r1"),
		Operation: opencdc.OperationCreate,
		Metadata:  map[string]string{"foo1": "bar1"},
		Key:       opencdc.RawData("secret-key"),
		Payload: opencdc.Change{
			After: opencdc.StructuredData{"ssn": "123-45-6789", "card": "4111111111111111"},
		},
	}}
}

// TestDestination_Write_DefaultDoesNotLogPayload is the regression test for
// GHSA-5c9c-5rf5-2r5q: by default neither key nor payload may reach the log.
func TestDestination_Write_DefaultDoesNotLogPayload(t *testing.T) {
	is := is.New(t)

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	ctx := logger.WithContext(context.Background())

	var dest Destination
	is.NoErr(sdk.Util.ParseConfig(ctx, map[string]string{"level": "warn"},
		dest.Config(), Connector.NewSpecification().DestinationParams))
	is.NoErr(dest.Open(ctx))

	_, err := dest.Write(ctx, newTestRecords())
	is.NoErr(err)

	out := buf.String()
	for _, leaked := range []string{"123-45-6789", "4111111111111111", "secret-key", "c2VjcmV0LWtleQ=="} {
		is.True(!strings.Contains(out, leaked)) // default output must not contain record data
	}
	is.True(strings.Contains(out, `"position":"r1"`) || strings.Contains(out, `"position":"cjE="`))
	is.True(strings.Contains(out, `"foo1":"bar1"`))
	sum := sha256.Sum256([]byte("secret-key"))
	is.True(strings.Contains(out, hex.EncodeToString(sum[:])))
}

func TestDestination_Write_PayloadOptIn(t *testing.T) {
	is := is.New(t)

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	ctx := logger.WithContext(context.Background())

	var dest Destination
	is.NoErr(sdk.Util.ParseConfig(ctx, map[string]string{"level": "warn", "payload": "true"},
		dest.Config(), Connector.NewSpecification().DestinationParams))
	is.NoErr(dest.Open(ctx))

	_, err := dest.Write(ctx, newTestRecords())
	is.NoErr(err)
	is.True(strings.Contains(buf.String(), "123-45-6789"))
}
