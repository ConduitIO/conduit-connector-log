# Conduit Connector Log

A [Conduit](https://conduit.io) destination connector that simply logs records.

## How to build?

Run `make build` to build the connector.

## Testing

Run `make test` to run all the unit tests.

## Destination

The destination connector logs records using the built-in Conduit logger.

### Configuration

| name    | description                                    | required | default value |
|---------|------------------------------------------------|----------|---------------|
| `level` | Log level (ERROR, WARN, INFO, DEBUG or TRACE). | false    | INFO          |

