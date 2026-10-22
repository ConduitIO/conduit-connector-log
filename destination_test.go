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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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
// GHSA-5c9c-5rf5-2r5q: by default neither key, payload, the raw position nor a
// free-form metadata value may reach the log.
func TestDestination_Write_DefaultDoesNotLogPayload(t *testing.T) {
	is := is.New(t)

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	ctx := logger.WithContext(context.Background())

	var dest Destination
	is.NoErr(sdk.Util.ParseConfig(ctx, map[string]string{"level": "warn"},
		dest.Config(), Connector.NewSpecification().DestinationParams))
	is.NoErr(dest.Open(ctx))

	recs := newTestRecords()
	recs[0].Position = []byte("pos-SECRET-POSITION")
	recs[0].Metadata = map[string]string{
		"foo1":                        "bar-SECRET-META",
		"opencdc.readAt":              "1700000000",
		"opencdc.collection":          "users",
		"conduit.source.connector.id": "pipeline:src",
		"conduit.dlq.nack.error":      `strconv.Atoi: parsing "SECRET-NUM-789": invalid syntax`,
		"conduit.dlq.nack.node.id":    "n1",
		"opencdc.file.name":           "SECRET-FILE-NAME",
		"kafka.header.token":          "SECRET-TOKEN",
	}
	_, err := dest.Write(ctx, recs)
	is.NoErr(err)

	out := buf.String()
	for _, leaked := range []string{
		"123-45-6789", "4111111111111111", "secret-key", "c2VjcmV0LWtleQ==",
		"SECRET-POSITION", base64.StdEncoding.EncodeToString(recs[0].Position),
		"SECRET-META", "SECRET-NUM-789", "SECRET-FILE-NAME", "SECRET-TOKEN",
	} {
		is.True(!strings.Contains(out, leaked)) // default output must not contain record data
	}
	// the safe identifiers are logged as they are
	for _, kept := range []string{`"opencdc.readAt":"1700000000"`, `"opencdc.collection.hash":"` + saltedHash([]byte("users")) + `"`, `"conduit.source.connector.id":"pipeline:src"`, `"conduit.dlq.nack.node.id":"n1"`} {
		is.True(strings.Contains(out, kept))
	}
	// the nack error: the Go qualifier only, plus a hash
	is.True(strings.Contains(out, `"conduit.dlq.nack.error":"strconv.Atoi"`))
	is.True(strings.Contains(out, `"dropped":3`)) // foo1, opencdc.file.name, kafka.header.token
}

func TestDestination_Write_NackErrorWithoutQualifierIsRedacted(t *testing.T) {
	is := is.New(t)
	var buf bytes.Buffer
	ctx := zerolog.New(&buf).WithContext(context.Background())
	var dest Destination
	is.NoErr(sdk.Util.ParseConfig(ctx, map[string]string{"level": "warn"}, dest.Config(), Connector.NewSpecification().DestinationParams))
	is.NoErr(dest.Open(ctx))

	long := "strconv." + strings.Repeat("A", 80) + ": SECRET-NUM-789"
	for _, e := range []string{
		"SECRET-NUM-789: not found", "failed to parse SECRET-NUM-789", "SECRET-NUM-789",
		"abc.Def: SECRET-NUM-789",    // the shape of a qualifier, but not a known package
		"user.Alice: SECRET-NUM-789", // an attacker-chosen value shaped like one
		long,                         // a known package, but beyond the length cap
	} {
		buf.Reset()
		_, err := dest.Write(ctx, []opencdc.Record{{Position: []byte("p"), Metadata: map[string]string{"conduit.dlq.nack.error": e}}})
		is.NoErr(err)
		is.True(!strings.Contains(buf.String(), "SECRET-NUM-789"))
		is.True(strings.Contains(buf.String(), `"conduit.dlq.nack.error":"redacted"`))
	}
}

// Hashes are salted: they are not the plain SHA-256 an attacker can compute
// from a guessed key or position, they are stable inside one process (so
// records can be correlated) and the salt is per process.
func TestDestination_Write_HashesAreSalted(t *testing.T) {
	is := is.New(t)
	var buf bytes.Buffer
	ctx := zerolog.New(&buf).WithContext(context.Background())
	var dest Destination
	is.NoErr(sdk.Util.ParseConfig(ctx, map[string]string{"level": "warn"}, dest.Config(), Connector.NewSpecification().DestinationParams))
	is.NoErr(dest.Open(ctx))

	buf.Reset() // drop the SDK's config debug lines
	rec := opencdc.Record{Position: []byte("p1"), Key: opencdc.RawData("secret-key")}
	_, err := dest.Write(ctx, []opencdc.Record{rec, rec})
	is.NoErr(err)
	out := buf.String()
	plainKey := sha256.Sum256([]byte("secret-key"))
	plainPos := sha256.Sum256([]byte("p1"))
	is.True(!strings.Contains(out, hex.EncodeToString(plainKey[:])))
	is.True(!strings.Contains(out, hex.EncodeToString(plainPos[:])))

	lines := strings.Split(strings.TrimSpace(out), "\n")
	is.Equal(len(lines), 2)
	is.Equal(lines[0], lines[1]) // stable within the process
	is.True(strings.Contains(lines[0], `"key_hash":"`+saltedHash([]byte("secret-key"))+`"`))
	is.True(strings.Contains(lines[0], `"position_hash":"`+saltedHash([]byte("p1"))+`"`))

	// another salt gives other hashes
	old := salt
	defer func() { salt = old }()
	salt = bytes.Repeat([]byte{7}, 32)
	is.True(saltedHash([]byte("p1")) != hex.EncodeToString(plainPos[:]))
	is.True(saltedHash([]byte("p1")) != hmacWith(old, []byte("p1")))
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

func hmacWith(key, data []byte) string {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return hex.EncodeToString(m.Sum(nil))
}

// The collection and the timestamps can be set by a processor from the payload.
// The collection is only ever logged hashed, and a timestamp only if it is an
// integer.
func TestDestination_Write_PayloadDerivedMetadataIsNotLogged(t *testing.T) {
	is := is.New(t)
	var buf bytes.Buffer
	ctx := zerolog.New(&buf).WithContext(context.Background())
	var dest Destination
	is.NoErr(sdk.Util.ParseConfig(ctx, map[string]string{"level": "warn"}, dest.Config(), Connector.NewSpecification().DestinationParams))
	is.NoErr(dest.Open(ctx))
	buf.Reset()

	_, err := dest.Write(ctx, []opencdc.Record{{Position: []byte("p"), Metadata: map[string]string{
		"opencdc.collection": "SECRET-COLLECTION-4111",
		"opencdc.createdAt":  "SECRET-CREATED-AT",
		"opencdc.readAt":     "1700000000123",
	}}})
	is.NoErr(err)
	out := buf.String()
	for _, leaked := range []string{"SECRET-COLLECTION-4111", "SECRET-CREATED-AT"} {
		is.True(!strings.Contains(out, leaked))
	}
	is.True(strings.Contains(out, `"opencdc.collection.hash":"`+saltedHash([]byte("SECRET-COLLECTION-4111"))+`"`))
	is.True(strings.Contains(out, `"opencdc.createdAt.hash":"`+saltedHash([]byte("SECRET-CREATED-AT"))+`"`))
	is.True(strings.Contains(out, `"opencdc.readAt":"1700000000123"`)) // an integer is fine
}
