# NPM dependencies metadata assembler

This program ingests a `package-lock.json` and outputs a payload with the following shape:

```json
{
    "analysed": "<package-lock.json content base64 encoded>",
    "packages": {
        "<package-name>": {
            "name": "<package-name>",
            "version": "<package-version>",
            "shasum": "<package-shasum>"
        }
    }
}
```

## Build

```bash
go build -o cli
```

## Run

```bash
./cli -i ./dev/package-lock.json -o payload.json -dry
```