# Conduit Connector Log

A [Conduit](https://conduit.io) destination connector that simply logs records.

<!-- readmegen:description -->
## Destination

The destination connector logs records using the built-in Conduit logger.

Example log message:

```
{"level":"info","record":{"position":"cjI=","operation":"create","metadata":{"foo":"bar","opencdc.version":"v1"},"key":{"my-id-field":1},"payload":{"before":nil,"after":{"my-payload-field":false}}}}
```

Note that the `position` field is base64 encoded, same goes for a key or payload
that contains raw byte data.

Keep in mind that Conduit's log level needs to be configured lower or equal to
the log level of the connector in order for the records to show up in the logs.
<!-- /readmegen:description -->

## Configuration

<!-- readmegen:destination.parameters.yaml -->
```yaml
version: 2.2
pipelines:
  - id: example
    status: running
    connectors:
      - id: example
        plugin: "log"
        settings:
          # The log level used to log records.
          # Type: string
          # Required: no
          level: "info"
          # Optional message that should be added to the log output of every
          # record.
          # Type: string
          # Required: no
          message: ""
          # Maximum delay before an incomplete batch is written to the
          # destination.
          # Type: duration
          # Required: no
          sdk.batch.delay: "0"
          # Maximum size of batch before it gets written to the destination.
          # Type: int
          # Required: no
          sdk.batch.size: "0"
          # Allow bursts of at most X records (0 or less means that bursts are
          # not limited). Only takes effect if a rate limit per second is set.
          # Note that if `sdk.batch.size` is bigger than `sdk.rate.burst`, the
          # effective batch size will be equal to `sdk.rate.burst`.
          # Type: int
          # Required: no
          sdk.rate.burst: "0"
          # Maximum number of records written per second (0 means no rate
          # limit).
          # Type: float
          # Required: no
          sdk.rate.perSecond: "0"
          # The format of the output record. See the Conduit documentation for a
          # full list of supported formats
          # (https://conduit.io/docs/using/connectors/configuration-parameters/output-format).
          # Type: string
          # Required: no
          sdk.record.format: "opencdc/json"
          # Options to configure the chosen output record format. Options are
          # normally key=value pairs separated with comma (e.g.
          # opt1=val2,opt2=val2), except for the `template` record format, where
          # options are a Go template.
          # Type: string
          # Required: no
          sdk.record.format.options: ""
          # Whether to extract and decode the record key with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.key.enabled: "true"
          # Whether to extract and decode the record payload with a schema.
          # Type: bool
          # Required: no
          sdk.schema.extract.payload.enabled: "true"
```
<!-- /readmegen:destination.parameters.yaml -->

## How to build?

Run `make build` to build the connector.

## Testing

Run `make test` to run all the unit tests.
