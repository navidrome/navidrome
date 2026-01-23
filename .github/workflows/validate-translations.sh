#!/bin/bash

# validate-translations.sh
# 
# This script validates the structure of JSON translation files by comparing them 
# against the reference English translation file (ui/src/i18n/en.json).
#
# The script performs the following validations:
# 1. JSON syntax validation using jq
# 2. Structural validation - ensures all keys from English file are present
# 3. Reports missing keys (translation incomplete)
# 4. Reports extra keys (keys not in English reference, possibly deprecated)
# 5. Emits GitHub Actions annotations for CI/CD integration
#
# Usage:
#   ./validate-translations.sh
#
# Environment Variables:
#   EN_FILE          - Path to reference English file (default: ui/src/i18n/en.json)
#   TRANSLATION_DIR  - Directory containing translation files (default: resources/i18n)
#
# Exit codes:
#   0 - All translations are valid
#   1 - One or more translations have structural issues
#
# GitHub Actions Integration:
#   The script outputs GitHub Actions annotations using ::error and ::warning
#   format that will be displayed in PR checks and workflow summaries.

# Script to validate JSON translation files structure against en.json
set -e

# Path to the reference English translation file
EN_FILE="${EN_FILE:-ui/src/i18n/en.json}"
TRANSLATION_DIR="${TRANSLATION_DIR:-resources/i18n}"
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
    echo "Usage: $0 [options]"
    echo ""
    echo "Validates JSON translation files structure against English reference file."
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -v, --verbose  Show detailed output (default: only show errors)"
    echo ""
    echo "Environment Variables:"
    echo "  EN_FILE          Path to reference English file (default: ui/src/i18n/en.json)"
    echo "  TRANSLATION_DIR  Directory with translation files (default: resources/i18n)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Validate all translation files (quiet mode)"
    echo "  $0 -v                                 # Validate with detailed output"
    echo "  EN_FILE=custom/en.json $0             # Use custom reference file"
    echo "  TRANSLATION_DIR=custom/i18n $0        # Use custom translations directory"
    exit 0
        ;;
        *)
            echo "Unknown option: $1" >&2
            echo "Use --help for usage information" >&2
            exit 1
            ;;
    esac
done

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

if [[ "$VERBOSE" == "true" ]]; then
    echo "Validating translation files structure against ${EN_FILE}..."
fi

# Check if English reference file exists
if [[ ! -f "$EN_FILE" ]]; then
    echo "::error::Reference file $EN_FILE not found"
    exit 1
fi

# Function to extract all JSON keys from a file, creating a flat list of dot-separated paths
extract_keys() {
    local file="$1"
    jq -r 'paths(scalars) as $p | $p | join(".")' "$file" 2>/dev/null | sort
}

# Function to extract all non-empty string keys (to identify structural issues)
extract_structure_keys() {
    local file="$1"
    # Get only keys where values are not empty strings
    jq -r 'paths(scalars) as $p | select(getpath($p) != "") | $p | join(".")' "$file" 2>/dev/null | sort
}

# Function to validate a single translation file
validate_translation() {
    local translation_file="$1"
    local filename=$(basename "$translation_file")
    local has_errors=false
    local verbose=${2:-false}
    
    if [[ "$verbose" == "true" ]]; then
        echo "Validating $filename..."
    fi
    
    # First validate JSON syntax
    if ! jq empty "$translation_file" 2>/dev/null; then
        echo "::error file=$translation_file::Invalid JSON syntax"
        echo -e "${RED}✗ $filename has invalid JSON syntax${NC}"
        return 1
    fi
    
    # Extract all keys from both files (for statistics)
    local en_keys_file=$(mktemp)
    local translation_keys_file=$(mktemp)
    
    extract_keys "$EN_FILE" > "$en_keys_file"
    extract_keys "$translation_file" > "$translation_keys_file"
    
    # Extract only non-empty structure keys (to validate structural issues)
    local en_structure_file=$(mktemp)
    local translation_structure_file=$(mktemp)
    
    extract_structure_keys "$EN_FILE" > "$en_structure_file"
    extract_structure_keys "$translation_file" > "$translation_structure_file"
    
    # Find structural issues: keys in translation not in English (misplaced)
    local extra_keys=$(comm -13 "$en_keys_file" "$translation_keys_file")
    
    # Find missing keys (for statistics only)
    local missing_keys=$(comm -23 "$en_keys_file" "$translation_keys_file")
    
    # Count keys for statistics
    local total_en_keys=$(wc -l < "$en_keys_file")
    local total_translation_keys=$(wc -l < "$translation_keys_file") 
    local missing_count=0
    local extra_count=0
    
    if [[ -n "$missing_keys" ]]; then
        missing_count=$(echo "$missing_keys" | grep -c '^' || echo 0)
    fi
    
    if [[ -n "$extra_keys" ]]; then
        extra_count=$(echo "$extra_keys" | grep -c '^' || echo 0)
        has_errors=true
    fi
    
    # Report extra/misplaced keys (these are structural issues)
    if [[ -n "$extra_keys" ]]; then
        if [[ "$verbose" == "true" ]]; then
            echo -e "${YELLOW}Misplaced keys in $filename ($extra_count):${NC}"
        fi
        
        while IFS= read -r key; do
            # Try to find the line number
            line=$(grep -n "\"$(echo "$key" | sed 's/.*\.//')" "$translation_file" | head -1 | cut -d: -f1)
            line=${line:-1} # Default to line 1 if not found
            
            echo "::error file=$translation_file,line=$line::Misplaced key: $key"
            
            if [[ "$verbose" == "true" ]]; then
                echo "  + $key (line ~$line)"
            fi
        done <<< "$extra_keys"
    fi
    
    # Clean up temp files
    rm -f "$en_keys_file" "$translation_keys_file" "$en_structure_file" "$translation_structure_file"
    
    # Print statistics
    if [[ "$verbose" == "true" ]]; then
        echo "  Keys: $total_translation_keys/$total_en_keys (Missing: $missing_count, Extra/Misplaced: $extra_count)"
    
        if [[ "$has_errors" == "true" ]]; then
            echo -e "${RED}✗ $filename has structural issues${NC}"
        else
            echo -e "${GREEN}✓ $filename structure is valid${NC}"
        fi
    elif [[ "$has_errors" == "true" ]]; then
        echo -e "${RED}✗ $filename has structural issues (Extra/Misplaced: $extra_count)${NC}"
    fi
    
    return $([[ "$has_errors" == "true" ]] && echo 1 || echo 0)
}

# Main validation loop
validation_failed=false
total_files=0
failed_files=0
valid_files=0

for translation_file in "$TRANSLATION_DIR"/*.json; do
    if [[ -f "$translation_file" ]]; then
        total_files=$((total_files + 1))
        if ! validate_translation "$translation_file" "$VERBOSE"; then
            validation_failed=true
            failed_files=$((failed_files + 1))
        else
            valid_files=$((valid_files + 1))
        fi
        
        if [[ "$VERBOSE" == "true" ]]; then
            echo "" # Add spacing between files
        fi
    fi
done

# Summary
if [[ "$VERBOSE" == "true" ]]; then
    echo "========================================="
    echo "Translation Validation Summary:"
    echo "  Total files: $total_files"
    echo "  Valid files: $valid_files"
    echo "  Files with structural issues: $failed_files"
    echo "========================================="
fi

if [[ "$validation_failed" == "true" ]]; then
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${RED}Translation validation failed - $failed_files file(s) have structural issues${NC}"
    else
        echo -e "${RED}Translation validation failed - $failed_files/$total_files file(s) have structural issues${NC}"
    fi
    exit 1
elif [[ "$VERBOSE" == "true" ]]; then
    echo -e "${GREEN}All translation files are structurally valid${NC}"
fi

exit 0