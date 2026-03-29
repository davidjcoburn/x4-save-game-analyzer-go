# X4 Save Game Analyzer (Go)

A Go-based tool for analyzing X4: Foundations save games.

## Features

- High-performance streaming XML parsing.
- Memory efficient (processes large saves without loading the full XML into memory).
- Supports `.xml.gz` and `.xml` formats.
- Scans for:
    - Kha'ak Hives and Nests.
    - Unowned (Abandoned) Ships.
    - Data Vaults (Decrypted status and locations).
    - Ship search by name/macro/code.

## Usage

### Build

```bash
go build -o x4-analyzer ./cmd/analyzer
```

### Run

```bash
./x4-analyzer -path "path/to/save.xml.gz" [flags]
```

### Flags

- `-path`: Path to the X4 save game file.
- `-ship`: Search for a ship by name.
- `-khaak`: Scan for Kha'ak targets.
- `-unowned`: Search for unowned/abandoned ships.
- `-vaults`: List all Data Vaults.

## Project Structure

- `cmd/analyzer/`: Entry point for the CLI tool.
- `internal/analyzer/`: Core parsing and analysis logic.
- `macro_map.json`: Data file for mapping internal macros to human-readable names.
- `TECHNICAL_SPEC.md`: Detailed technical specification.

## Performance

This tool is designed for speed and low memory usage, capable of processing large save files in seconds.
