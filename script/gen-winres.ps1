#!/usr/bin/env pwsh

# Generate Windows resource files as '.syso'
# 
# Usage:
#   gen-winres.ps1 <arch> <version> <path-to-winres.json> <output-path>
#
# Arguments:
#   <arch>                 comma-separated list of architectures (e.g. "386,amd64,arm64")
#   <version>              version string (e.g. "1.0.0")
#   <path-to-winres.json>  path to the `winres.json` file containing static metadata
#   <output-path>          directory where the generated `.syso` files should be placed
#
# The created `.syso` files are named as `rsrc_windows_<arch>.syso` which helps
# Go compiler to pick the correct file based on the target architecture.
#

$ErrorActionPreference = "Stop"

$_arch = $args[0]
if ([string]::IsNullOrEmpty($_arch)) {
    Write-Host "error: architecture argument is missing"
    exit 1
}

$_version = $args[1]
if ([string]::IsNullOrEmpty($_version)) {
    Write-Host "error: version argument is missing"
    exit 1
}

$_winresJson = $args[2]
if ([string]::IsNullOrEmpty($_winresJson)) {
    Write-Host "error: path to winres.json is missing"
    exit 1
}

if (-not (Test-Path $_winresJson)) {
    Write-Host "error: winres.json file not found at '$_winresJson'"
    exit 1
}

$_output = $args[3]
if ([string]::IsNullOrEmpty($_output)) {
    Write-Host "error: output path is missing"
    exit 1
}

if (-not (Test-Path $_output -PathType Container)) {
    Write-Host "error: output path '$_output' is not a directory"
    exit 1
}

# Note that we intentionally leave the `--file-version` option in the command
# below, because it's meant to be a 4-component version, while ours is a semver
# (3-component). If we populate the `--file-version` with our semver value, then
# a zero component will be added to the end, which is not what we want.

go run github.com/tc-hib/go-winres@v0.3.3 make `
    --arch "$_arch" `
    --product-version "$_version" `
    --in "$_winresJson" `
    --out rsrc

Move-Item -Path ".\rsrc_*.syso" -Destination "$_output" -Force
