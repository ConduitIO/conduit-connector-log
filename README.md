# Conduit Connector Log

A [Conduit](https://conduit.io) destination connector that simply logs records.

## How to build?

Run `make build` to build the connector.

## Testing

Run `make test` to run all the unit tests.

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

### Configuration

| name      | description                                                              | required | default value |
|-----------|--------------------------------------------------------------------------|----------|---------------|
| `level`   | Log level (error, warn, info, debug or trace).                           | false    | info          |
| `message` | Optional message that should be added to the log output of every record. | false    |               |
