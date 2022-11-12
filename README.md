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

If you have a `package.json` but not a `package-lock.json`, run:

```bash
./cli
```

If you want to use an already present `package-lock.json`, run:

```bash
./cli -i ./package-lock.json
```

### TODOs

- ignore `file:` dependencies