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
	"testing"

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
		"ERROR": zerolog.ErrorLevel,
		"WARN":  zerolog.WarnLevel,
		"INFO":  zerolog.InfoLevel,
		"DEBUG": zerolog.DebugLevel,
		"TRACE": zerolog.TraceLevel,
	}

	for have, want := range testCases {
		t.Run(have, func(t *testing.T) {
			var dest Destination
			err := dest.Configure(ctx, map[string]string{"level": have})
			is.NoErr(err)
			is.Equal(dest.level, want)
		})
	}
}

func TestDestination_Configure_Fail(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	var dest Destination
	err := dest.Configure(ctx, map[string]string{"level": "invalid"})
	is.True(err != nil)
}

func TestDestination_Configure_Write(t *testing.T) {
	is := is.New(t)

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	ctx := logger.WithContext(context.Background()) // sdk takes logger from the context

	var dest Destination

	err := dest.Configure(ctx, map[string]string{"level": "warn"})
	is.NoErr(err)

	have := []sdk.Record{{
		Position:  []byte("r1"),
		Operation: sdk.OperationCreate,
		Metadata:  map[string]string{"foo1": "bar1"},
		Key:       sdk.RawData("raw-key"),
		Payload: sdk.Change{
			Before: nil,
			After:  sdk.RawData("raw-payload"),
		},
	}, {
		Position:  []byte("r2"),
		Operation: sdk.OperationUpdate,
		Metadata:  map[string]string{"foo2": "bar2"},
		Key:       sdk.StructuredData{"r2-key": 1},
		Payload: sdk.Change{
			Before: sdk.StructuredData{"r2-payload": true},
			After:  sdk.StructuredData{"r2-payload": false},
		},
	}}

	want := []string{
		`{"level":"warn","record":{"position":"cjE=","operation":"create","metadata":{"foo1":"bar1","opencdc.version":"v1"},"key":"cmF3LWtleQ==","payload":{"before":null,"after":"cmF3LXBheWxvYWQ="}}}`,
		`{"level":"warn","record":{"position":"cjI=","operation":"update","metadata":{"foo2":"bar2","opencdc.version":"v1"},"key":{"r2-key":1},"payload":{"before":{"r2-payload":true},"after":{"r2-payload":false}}}}`,
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
