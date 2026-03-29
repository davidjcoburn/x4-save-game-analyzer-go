# Technical Specification: X4 Save Game Analyzer

## 1. System Overview
The **X4 Save Game Analyzer** is a high-performance ETL (Extract, Transform, Load) tool developed in Python. It is designed to process large, Gzip-compressed XML save files from Egosoft's *X4: Foundations*. The tool enables users to extract strategic intelligence—such as abandoned ship locations, Kha'ak installations, and Data Vault status—without loading the game client.

### Key Technologies
*   **Language:** Python 3.10+
*   **XML Engine:** `lxml.etree.iterparse` (C-based streaming parser)
*   **Compression:** `gzip` (Standard library)
*   **UI/UX:** `tqdm` (CLI Progress), `tkinter` (Native File Dialog)

---

## 2. Architectural Components

### 2.1 Entry Point (`main.py`)
The orchestrator responsible for user interaction and data presentation.
*   **CLI Parser:** Utilizes `argparse` to handle scanning flags (`--ship`, `--khaak`, `--unowned`, `--vaults`).
*   **Interactive Mode:** Automatically triggers if no CLI arguments are provided, using `tkinter` for cross-platform file selection.
*   **Data Formatting:** Implements conversion logic for game-specific units:
    *   **Credits:** Converts centicredits to Credits ($1/100$ scaling).
    *   **Time:** Converts raw seconds into a `Dd Hh Mm Ss` format.
    *   **Coordinates:** Converts meters to kilometers for spatial reporting.

### 2.2 Core Scanner (`src/analyzer/parser.py`)
A memory-efficient, single-pass parser utilizing a "pull" XML model.
*   **Memory Management:** Implements an aggressive tree-clearing strategy. By calling `elem.clear()` and removing preceding siblings during the `end` event of a `<component>`, the tool maintains a near-constant memory footprint regardless of XML size.
*   **Hierarchy Resolution:** Maintains a flat hash map (`id_to_info`) of all component IDs to their parents and local offsets. This allows the system to resolve global sector coordinates post-parse.
*   **Streaming Progress:** Features a `ProgressWrapper` class that wraps the `gzip` file object, allowing `tqdm` to track the actual bytes consumed by the XML parser in real-time.

### 2.3 Constants & Configuration (`src/analyzer/constants.py`)
Centralizes business logic and game-specific identifiers.
*   **Scaling Constants:** Defines standard conversion factors (e.g., `METER_TO_KM = 1000.0`).
*   **Class Filters:** Uses Python `Set` objects for $O(1)$ lookup performance when filtering component classes (e.g., `SHIP_CLASSES`, `STATION_CLASSES`).
*   **Kha'ak Logic:** Stores substrings used to identify specific Kha'ak station types (Hives vs. Nests).

---

## 3. Data Flow & Algorithms

### 3.1 Extraction Logic
1.  **Initialization:** Loads `macro_map.json` to resolve internal macro IDs to localized names.
2.  **Event Loop:** Iterates through XML nodes. When a `<component>` is encountered:
    *   Local coordinates are extracted from the `<offset><position>` sub-tags.
    *   The component is registered in the hierarchy map.
    *   If the component matches active search criteria (e.g., `owner="ownerless"`), it is added to the results buffer.
3.  **Cleanup:** Once the stream ends, the parser iterates through found results to resolve their final sector and system locations.

### 3.2 Global Coordinate Calculation
Coordinates in X4 are stored relative to the immediate parent. The analyzer calculates the global sector position using a recursive summation:
$$Pos_{global} = \sum_{i=0}^{n} Pos_{local\_i}$$
where $n$ represents the depth of the component within the sector hierarchy.

---

## 4. Performance & Scalability
*   **Time Complexity:** $O(N)$ where $N$ is the number of XML nodes.
*   **Space Complexity:** $O(M)$ where $M$ is the number of unique component IDs (significantly smaller than the full XML DOM).
*   **Efficiency:** Capable of processing 1GB+ XML files in seconds, maintaining low RAM usage (typically <250MB) by avoiding full document tree construction.

---

## 5. Maintenance & Supportability
*   **Extensibility:** New scan types can be added by updating the `SHIP_CLASSES` or `STATION_CLASSES` sets in `constants.py` and adding a filter condition in `parser.py`.
*   **Decoupled Data:** Uses an external `macro_map.json` for name resolution, allowing the tool to support new DLCs or mods without code changes.
*   **Verification:** Includes a `unittest` suite (`tests/test_parser.py`) that uses mock objects to simulate Gzip streams, ensuring core logic remains valid during refactoring.
