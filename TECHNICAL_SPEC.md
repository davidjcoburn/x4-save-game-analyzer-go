# Technical Specification: X4 Save Game Analyzer (Go)

## 1. System Overview
The **X4 Save Game Analyzer** is a high-performance ETL (Extract, Transform, Load) tool developed in Go. It is designed to process large, Gzip-compressed XML save files from Egosoft's *X4: Foundations*. The tool enables users to extract strategic intelligence—such as abandoned ship locations, Kha'ak installations, and Data Vault status—without loading the game client.

### Key Technologies
*   **Language:** Go (Golang) 1.21+
*   **XML Engine:** `encoding/xml` (Streaming `xml.Decoder`)
*   **Compression:** `compress/gzip` (Standard library)
*   **UI/UX:** CLI with platform-specific native file dialogs (PowerShell on Windows)
*   **Static Assets:** `go:embed` for internal data files

---

## 2. Architectural Components

### 2.1 Entry Point (`cmd/analyzer/main.go`)
The orchestrator responsible for user interaction and data presentation.
*   **Flag Parser:** Utilizes the standard `flag` package to handle scanning flags (`-ship`, `-khaak`, `-unowned`, `-vaults`, `-path`).
*   **Interactive Mode:** Automatically triggers if no `-path` is provided, using a platform-specific file dialog.
*   **Platform-Specific Dialogs:** Uses build tags (`//go:build windows` and `//go:build !windows`) to provide a native OpenFileDialog on Windows via PowerShell, while remaining portable.
*   **Data Formatting:** Implements conversion logic for game-specific units in `internal/analyzer/utils.go`:
    *   **Credits:** Converts centicredits to Credits ($1/100$ scaling).
    *   **Time:** Converts raw seconds into a `Dd Hh Mm Ss` format.
    *   **Coordinates:** Converts meters to kilometers for spatial reporting.

### 2.2 Core Scanner (`internal/analyzer/parser.go`)
A memory-efficient, streaming parser utilizing a "pull" XML model.
*   **Memory Management:** Uses `xml.Decoder` to iterate through tokens without loading the full XML DOM. It maintains a near-constant memory footprint regardless of the XML size.
*   **Hierarchy Resolution:** Maintains a map (`idToInfo`) of component IDs to their parents, classes, and local positions. For memory efficiency, it only stores components necessary for hierarchy tracking (clusters, sectors, and their parents).
*   **Macro Resolution:** Uses an embedded `macro_map.json` (via `go:embed`) to resolve internal macro names to human-readable labels.

### 2.3 Macro Map Builder (`internal/analyzer/build_map.go`)
An optional utility to rebuild the internal macro database from game data.
*   **Efficiency:** Uses a single-pass `filepath.Walk` to scan game data directories for both translation files (`0001.xml`) and macro definition files.
*   **Resolution:** Recursively resolves nested translation tags (e.g., `{20001, 101}`).

---

## 3. Data Flow & Algorithms

### 3.1 Extraction Logic
1.  **Initialization:** Loads the macro map from either a local file or the embedded default.
2.  **Streaming Loop:** Iterates through XML tokens. When a `<component>` start element is encountered:
    *   Attributes (ID, class, macro, owner, name, code) are extracted.
    *   The component is pushed onto a stack to track parent-child relationships.
    *   The component is registered in the hierarchy map.
    *   If the component matches active search criteria (e.g., `owner="ownerless"`), it is added to the results buffer.
3.  **Position Extraction:** When a `<position>` tag is found, its `x,y,z` values are associated with the current component in the hierarchy map.
4.  **Final Resolution:** Once the stream ends, the analyzer traces the parent hierarchy for each result to calculate global sector coordinates and identify the system/sector names.

### 3.2 Global Coordinate Calculation
Coordinates in X4 are stored relative to the immediate parent. The analyzer calculates the global sector position by summing local offsets up the hierarchy:
$$Pos_{global} = \sum_{i=0}^{n} Pos_{local\_i}$$
where $n$ represents the depth of the component within the sector hierarchy.

---

## 4. Performance & Scalability
*   **Time Complexity:** $O(N)$ where $N$ is the number of XML nodes.
*   **Space Complexity:** $O(M)$ where $M$ is the number of components tracked in the hierarchy map (significantly smaller than the full XML DOM).
*   **Efficiency:** Processes large compressed save files (100MB+ compressed, 1GB+ uncompressed) in seconds. Memory usage is stabilized by the streaming decoder.

---

## 5. Maintenance & Supportability
*   **Coding Standards:** Adheres to idiomatic Go standards, including full documentation for exported symbols and `go fmt` formatting.
*   **Extensibility:** New scan types can be added by updating `constants.go` and adding filter conditions in `parser.go`.
*   **Decoupled Data:** Macro resolution is decoupled from the main logic, allowing for easy updates via the `-build-map` flag if game data changes.
*   **Testing:** Includes a comprehensive test suite (`internal/analyzer/parser_test.go`) that validates hierarchy resolution and scanning logic against mock XML data.
