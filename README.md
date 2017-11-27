# mongotunnel

mongotunnel will act as a proxy for the mongo wire protocol, while providing
interfaces for intercepting and handling the wire protocol constructs as
they come through.

At the moment the bulk of the code is only there to handle parsing the
wire protocol.

The eventual goal is to be able to provide a proof-of-concept for adding
MongoDB wire compatibility to CockroachDB.

# TODO

## Cleanups

- Change all function `bytes.Buffer` to `io.Reader` or `io.Writer`
- Standardize context and logging functions (pass context through, etc.)