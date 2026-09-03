# Technical Specification: X4 Save Game Analyzer (Go)

## 1. System Overview
The **X4 Save Game Analyzer** is a high-performance ETL (Extract, Transform, Load) tool developed in Go. It is designed to process large, Gzip-compressed XML save files from Egosoft's *X4: Foundations*. The tool enables users to extract strategic intelligence—such as abandoned ship locations, Kha'ak installations, and Data Vault status—without loading the game client.

### Key Technologies
*   **Language:** Go (Golang) 1.21+
*   **XML Engine:** High-performance streaming byte-slice tokenizer with zero-allocation attribute scanning
*   **Compression:** `github.com/klauspost/compress/gzip` with buffered stream pipeline
*   **UI/UX:** CLI with platform-specific native file dialogs (PowerShell STA on Windows)
*   **Static Assets:** `go:embed` for internal data files

---

## 2. Architectural Components

### 2.1 Entry Point (`cmd/analyzer/main.go`)
The orchestrator responsible for user interaction and data presentation.
*   **Flag Parser:** Utilizes the standard `flag` package to handle scanning flags (`-ship`, `-khaak`, `-unowned`, `-vaults`, `-path`).
*   **Interactive Mode:** Automatically triggers if no `-path` is provided, using a platform-specific file dialog.
*   **Persistent Search Loop:** When searching for ships in interactive mode, the tool maintains a loop allowing repeated queries (Name, Macro, or ID) against in-memory data.
*   **Platform-Specific Dialogs:** Uses build tags (`//go:build windows` and `//go:build !windows`) to provide native OpenFileDialog and FolderBrowserDialog on Windows via PowerShell.
*   **Data Formatting:** Implements conversion logic for game-specific units in `internal/analyzer/utils.go`:
    *   **Credits:** Converts centicredits to Credits ($1/100$ scaling) with comma-separated thousands formatting.
    *   **Time:** Converts raw seconds into a `Dd Hh Mm Ss` format with proper zero-unit handling.
    *   **Coordinates:** Converts meters to kilometers for spatial reporting, avoiding negative zero artifacts.

### 2.2 Core Scanner (`internal/analyzer/parser.go`)
A memory-efficient, streaming parser utilizing a pull-based byte-slice model.
*   **Memory Management:** Iterates through tag slices with zero-allocation key parsing and dynamic hash table allocation to minimize GC overhead.
*   **Collection Strategy:** If a ship search is requested, the scanner collects all ships during the initial pass to enable near-instantaneous filtering in the interactive loop.
*   **Hierarchy Resolution:** Maintains a map (`idToInfo`) of component IDs to their parents, classes, and local positions with cycle protection.
    *   **Connection Offsets:** Correctly handles offsets in both `<connection>` and `<component>` tags using a tag stack.
    *   **Sector-Relative Coordinates:** Summation stops at the `sector` level to ensure coordinates match the in-game map.
*   **Macro Resolution:** Uses an embedded `macro_map.json` (via `go:embed`) or a local override to resolve internal macro names to human-readable labels.

### 2.3 Macro Map Builder (`internal/analyzer/build_map.go`)
A utility to rebuild the internal macro database from game data.
*   **Efficiency:** Uses a single-pass `filepath.WalkDir` to collect XML files, followed by a processing pass with a progress indicator.
*   **Data Sources:**
    *   **Translations:** Parses `0001.xml` files for localized strings.
    *   **Macros:** Parses component macro files for `identification` tags (supporting both localized references and literal names).
    *   **Wares:** Parses `wares.xml` to link ship macros to their market names (critical for story ships like the Hyperion/Trinity).
*   **Resolution:** Recursively resolves nested translation tags and strips redundant X4 metadata (voice hints, comment tags) using balanced parenthesis logic.

---

## 3. Data Flow & Algorithms

### 3.1 Extraction Logic
1.  **Initialization:** Loads the macro map from either a local file or the embedded default.
2.  **Streaming Loop:** Iterates through XML tokens. When a `<component>` start element is encountered:
    *   Attributes (ID, class, macro, owner, name, code) are extracted.
    *   A tag stack ensures `<position>` tags are attributed to the correct parent (connection vs component).
    *   If the component is "interesting" (ship, station, module), it is registered in the hierarchy map.
3.  **Position Extraction:** Offsets are accumulated using a `pendingPos` buffer that is flushed into components or cleared when connections end to prevent "offset leakage".
4.  **Final Resolution:** Traces the parent hierarchy for each result to calculate sector-relative coordinates and identify system names.

---

## 4. Performance & Scalability
*   **Time Complexity:** $O(N)$ where $N$ is the number of XML nodes.
*   **Space Complexity:** $O(M)$ where $M$ is the number of components tracked in the hierarchy map.
*   **Efficiency:** Processes 1GB+ uncompressed XML in ~20-25 seconds on modern hardware. Repeated ship searches in the interactive loop take <1ms.

---

## 5. Maintenance & Supportability
*   **Coding Standards:** Adheres to Go standards, including documentation for exported symbols and `go fmt`.
*   **Extensibility:** Manual overrides in `BuildMacroMap` allow for quick fixes to special story ships or mod-added content that breaks standard naming conventions.
*   **Testing:** Comprehensive test suite (`parser_test.go`, `build_map_test.go`, `utils_test.go`) validates:
    *   Scanning logic and criteria filtering.
    *   Hierarchy and connection offset summation.
    *   Offset leak prevention.
    *   Macro map resolution and string cleaning.
