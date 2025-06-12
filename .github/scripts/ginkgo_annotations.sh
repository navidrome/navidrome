#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <test log>" >&2
  exit 1
fi

log_file="$1"

# Strip ANSI color codes first, then process with AWK
sed 's/\[[0-9;]*m//g' "$log_file" | awk '
  BEGIN {
    processed_tests = ""
  }
  
  # Match the detailed failure format first: • [FAILED] (more specific line numbers)
  /• \[FAILED\]/ {
    # Get the test name from the next line
    getline
    test_name = $0
    
    # Look for the file path in the next line
    getline
    file_line = $0
    
    if (file_line ~ /\.go:[0-9]+/) {
      # Extract file and line number, trim whitespace
      gsub(/^[ \t]+|[ \t]+$/, "", file_line)
      split(file_line, parts, ":")
      file = parts[1]
      line_no = parts[2]
      
      # Trim workspace path if present
      ws=ENVIRON["GITHUB_WORKSPACE"]
      if (ws != "" && index(file, ws) == 1) {
        file=substr(file, length(ws)+2)
      }
      
      # Mark this test as processed to avoid duplicates (use test name as key)
      if (index(processed_tests, test_name) == 0) {
        processed_tests = processed_tests test_name "|"
        printf "::error file=%s,line=%s::%s\n", file, line_no, test_name
      }
    }
  }
  # Match the summary format: [FAIL] test name (only if not already processed)
  /^[ \t]*\[FAIL\]/ {
    name=$0
    sub(/^[ \t]*\[FAIL\][ \t]*/, "", name)
    
    # Only process if we haven'\''t already processed this test
    if (index(processed_tests, name) == 0) {
      getline
      file_line = $0
      
      if (file_line ~ /\.go:[0-9]+/) {
        # Extract file and line number, trim whitespace
        gsub(/^[ \t]+|[ \t]+$/, "", file_line)
        split(file_line, parts, ":")
        file = parts[1]
        line_no = parts[2]
        
        # Trim workspace path if present
        ws=ENVIRON["GITHUB_WORKSPACE"]
        if (ws != "" && index(file, ws) == 1) {
          file=substr(file, length(ws)+2)
        }
        
        processed_tests = processed_tests name "|"
        printf "::error file=%s,line=%s::%s\n", file, line_no, name
      }
    }
  }
'

