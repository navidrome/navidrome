# CI-only check for the #6276 follow-ups. Every #6276 spec must have run and passed: it fails on
# any missing, skipped or failed spec in the groups below, and writes a summary to the job.
param(
    [Parameter(Mandatory)][string]$TestOutcome,
    [Parameter(Mandatory)][string[]]$Reports
)
$ErrorActionPreference = 'Stop'

# group (container text, or a spec name) = expected number of specs
$groups = [ordered]@{
    'PlaylistsPathPatterns'   = 19  # conf: Windows/Unix pattern conversion
    'PlaylistsPath validation' = 7  # conf: conf.LoadFromFile
    'InPath'                  = 35  # core/playlists: PR #6281 specs + scanner-built folders
    'Scanner - PlaylistsPath' = 22  # scanner: real folders, phase 1 + phase 4, recovery
    "produces different hash when PlaylistsPath starts including the folder's playlists" = 1
    'keeps the hash of a folder without playlists when PlaylistsPath changes' = 1
}

$specs = @()
$skipped = @()
foreach ($r in $Reports) {
    if (-not (Test-Path $r)) { throw "Missing Ginkgo report: $r" }
    foreach ($suite in (Get-Content $r -Raw | ConvertFrom-Json)) {
        foreach ($spec in $suite.SpecReports) {
            if ($spec.LeafNodeType -ne 'It') { continue }
            $specs += $spec
            if ($spec.State -eq 'skipped') { $skipped += "$($spec.ContainerHierarchyTexts -join ' > ') > $($spec.LeafNodeText)" }
        }
    }
}

$problems = @()
$rows = @('| Group | Expected | Ran | Passed |', '|---|---|---|---|')
foreach ($g in $groups.Keys) {
    $inGroup = @($specs | Where-Object { $_.ContainerHierarchyTexts -contains $g -or $_.LeafNodeText -eq $g })
    $passed = @($inGroup | Where-Object { $_.State -eq 'passed' })
    foreach ($s in ($inGroup | Where-Object { $_.State -ne 'passed' })) {
        $problems += "$g > $($s.LeafNodeText): $($s.State)"
    }
    if ($inGroup.Count -ne $groups[$g]) { $problems += "$g : expected $($groups[$g]) specs, found $($inGroup.Count)" }
    $rows += "| $g | $($groups[$g]) | $($inGroup.Count) | $($passed.Count) |"
}
if ($TestOutcome -ne 'success') { $problems += "Test step outcome: $TestOutcome" }

$total = @($specs).Count
$summary = @("## #6276 follow-ups on $([System.Environment]::OSVersion.VersionString)", '',
    "All specs in conf, core/playlists, scanner: $total ($(@($specs | Where-Object State -eq 'passed').Count) passed, $($skipped.Count) skipped). Test step: **$TestOutcome**", '') + $rows
if ($skipped.Count -gt 0) {
    $summary += '', 'Skipped (pre-existing Windows skips, outside the #6276 groups):'
    $summary += @($skipped | ForEach-Object { "- $_" })
}
if ($problems.Count -eq 0) {
    $summary += '', '**Result: every #6276 spec ran and passed.**'
} else {
    $summary += '', '**Result: FAILED**'
    $summary += @($problems | ForEach-Object { "- $_" })
}
$summary -join "`n" | Tee-Object -Append -FilePath $env:GITHUB_STEP_SUMMARY
if ($problems.Count -gt 0) { exit 1 }
