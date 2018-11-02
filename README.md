# rfc2csv

Converts an IETF standards document (RFC) into a CSV file, ignoring common sections such as the `Introduction`, `Definitions`, `Acknowledgements`, etc.

This application was written with the intention to create a quick compliance check list from an RFC document.

## Install
```bash
go install github.com/digitorus/rfc2csv
```

## Usage
```bash
rfc2csv {rfc number}
rfc2csv {rfc number} {rfc number} {rfc number} ...

rfc2csv 5280
rfc2csv 5280 5019
```

## Using Docker
```bash
docker run -ti digitorus/rfc2csv sh
```
After this you can use the `rfc2csv` command from above.

## Reconnect Docker image
```bash
docker ps -a
docker start {container id}
docker attach {container id}
```