# Define base paths
$gameRoot = "E:\SteamLibrary\steamapps\common\X4 Foundations"
$extensionsDir = Join-Path $gameRoot "extensions"
$outputBase = "C:\Users\djcob\source\x4-save-game-analyzer\game-data"
$catTool = Join-Path $gameRoot "XRCatTool.exe"

# Ensure the cat tool exists
if (-not (Test-Path $catTool)) {
    Write-Error "XRCatTool.exe not found at $catTool"
    return
}

# Iterate through each folder in the extensions directory
Get-ChildItem -Path $extensionsDir -Directory | ForEach-Object {
    $extFolder = $_.Name
    $extPath = $_.FullName

    # Collect all .cat files in this extension folder
    $catFiles = Get-ChildItem -Path $extPath -Filter "*.cat" | ForEach-Object { $_.FullName }

    if ($catFiles.Count -gt 0) {
        $outputPath = Join-Path $outputBase $extFolder

        # Create output directory if it doesn't exist
        if (-not (Test-Path $outputPath)) {
            New-Item -ItemType Directory -Path $outputPath -Force | Out-Null
        }

        Write-Host "Processing extension: $extFolder"

        # Execute the tool
        # Using the call operator (&) to run the EXE with the array of cat files
        & $catTool -in $catFiles -out $outputPath -include "xml"
    }
    else {
        Write-Verbose "No .cat files found in $extFolder, skipping."
    }
}

Write-Host "Extraction complete."