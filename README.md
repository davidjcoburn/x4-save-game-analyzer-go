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
go build -o x4-analyzer.exe ./cmd/analyzer
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
- `-extract`: Extract XML game data from .cat files using XRCatTool.exe.
- `-game`: Path to X4 Foundations root directory (for extraction).
- `-build-map`: Rebuild internal macro map from extracted game-data.

## Getting Ship and Map Data

To resolve internal macro names (e.g., `ship_arg_l_destroyer_01_macro`) into human-readable names (e.g., `Behemoth Sentinel`), the analyzer needs a `macro_map.json` file. You can generate this file from your own game installation:

1.  **Download XRCatTool.exe:**
    -   Download the "X Catalog Tool" from the [Egosoft Bonus Downloads](https://www.egosoft.com/download/x4/bonus_en.php) page (requires login).
    -   Alternatively, install "X Tools" from your Steam Library (check "Tools" in the library filter).
    -   Place `XRCatTool.exe` in the same folder as the analyzer.
2.  **Extract Game Data:**
    -   Run the analyzer with the extract flag:
      ```bash
      ./x4-analyzer -extract
      ```
    -   You will be prompted to select your X4 Foundations root directory via a folder dialog.
    -   Alternatively, provide the path directly:
      ```bash
      ./x4-analyzer -extract -game "C:\Games\X4 Foundations"
      ```
    -   This will extract the necessary XML files into a `./game-data` folder.
3.  **Build the Macro Map:**
    -   Run the analyzer with the build-map flag:
      ```bash
      ./x4-analyzer -build-map
      ```
    -   This will scan the `./game-data` folder and generate a `macro_map.json`.
4.  **Usage:**
    -   The analyzer will now use this local `macro_map.json` to provide accurate names in all future scans.

## Interactive Mode

If no `-path` is provided, the tool enters interactive mode:
1.  **File Selection:** A native OpenFileDialog opens (Windows).
2.  **Criteria Menu:** A CLI menu allows selecting scan targets (Kha'ak, Unowned, Vaults, Ship Search).

- `cmd/analyzer/`: Entry point for the CLI tool.
- `internal/analyzer/`: Core parsing and analysis logic.
- `macro_map.json`: Data file for mapping internal macros to human-readable names.
- `TECHNICAL_SPEC.md`: Detailed technical specification.

## Performance

This tool is designed for speed and low memory usage, capable of processing large save files in seconds.
