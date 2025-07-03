#!/bin/bash

# Validate JSON translation files structure against English reference
# Usage: ./validate-translations.sh [-v|--verbose] [-h|--help]

set -e

# Default values
VERBOSE=false
EN_FILE="${EN_FILE:-ui/src/i18n/en.json}"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [-v|--verbose] [-h|--help]"
            echo ""
            echo "Validate JSON translation files structure against English reference"
            echo ""
            echo "Options:"
            echo "  -v, --verbose    Show detailed output including valid files"
            echo "  -h, --help       Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  EN_FILE          Path to English reference file (default: ui/src/i18n/en.json)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "::error::jq is required but not installed"
    exit 1
fi

# Check if English reference file exists
if [[ ! -f "$EN_FILE" ]]; then
    echo "::error::English reference file not found: $EN_FILE"
    exit 1
fi

# Function to get line number for a key path in JSON file
get_line_number() {
    local file="$1"
    local key_path="$2"
    
    # Convert key path to grep pattern (e.g., "song.fields.title" -> "title")
    local key_name
    key_name=$(echo "$key_path" | sed 's/.*\.//')
    
    # Find the line number of the key
    grep -n "\"$key_name\"" "$file" 2>/dev/null | head -1 | cut -d: -f1 || echo "1"
}

# Function to extract all keys from a JSON file
extract_keys() {
    local file="$1"
    jq -r 'paths(scalars) as $p | $p | join(".")' "$file" 2>/dev/null | sort
}

# Function to validate a single translation file
validate_translation() {
    local file="$1"
    local has_errors=false
    
    # Extract keys from both files
    local en_keys_file
    local trans_keys_file
    en_keys_file=$(mktemp)
    trans_keys_file=$(mktemp)
    
    extract_keys "$EN_FILE" > "$en_keys_file"
    extract_keys "$file" > "$trans_keys_file"
    
    local en_count
    local trans_count
    en_count=$(wc -l < "$en_keys_file")
    trans_count=$(wc -l < "$trans_keys_file")
    
    # Find missing keys (in English but not in translation)
    local missing_keys
    missing_keys=$(comm -23 "$en_keys_file" "$trans_keys_file")
    
    # Find extra keys (in translation but not in English)
    local extra_keys
    extra_keys=$(comm -13 "$en_keys_file" "$trans_keys_file")
    
    if [[ -n "$missing_keys" ]]; then
        has_errors=true
        while IFS= read -r key; do
            [[ -z "$key" ]] && continue
            local line_num
            line_num=$(get_line_number "$EN_FILE" "$key")
            echo "::error file=$file,line=$line_num::Missing translation key: $key"
        done <<< "$missing_keys"
    fi
    
    if [[ -n "$extra_keys" ]]; then
        while IFS= read -r key; do
            [[ -z "$key" ]] && continue
            local line_num
            line_num=$(get_line_number "$file" "$key")
            echo "::warning file=$file,line=$line_num::Extra translation key (not in English): $key"
        done <<< "$extra_keys"
    fi
    
    # Output results based on verbose mode
    if [[ "$VERBOSE" == "true" ]]; then
        if [[ "$has_errors" == "true" ]]; then
            echo "❌ $file: $trans_count/$en_count keys ($(echo "$missing_keys" | wc -w) missing)"
        else
            echo "✅ $file: $trans_count/$en_count keys (complete)"
        fi
    elif [[ "$has_errors" == "true" ]]; then
        echo "❌ $file: $trans_count/$en_count keys ($(echo "$missing_keys" | wc -w) missing)"
    fi
    
    # Cleanup
    rm -f "$en_keys_file" "$trans_keys_file"
    
    [[ "$has_errors" == "false" ]]
}

# Main validation loop
total_files=0
valid_files=0
has_any_errors=false

[[ "$VERBOSE" == "true" ]] && echo "Validating translation files against $EN_FILE..."
[[ "$VERBOSE" == "true" ]] && echo ""

for file in resources/i18n/*.json; do
    [[ ! -f "$file" ]] && continue
    
    # Skip if it's the English file itself
    if [[ "$(basename "$file")" == "$(basename "$EN_FILE")" ]]; then
        continue
    fi
    
    total_files=$((total_files + 1))
    
    [[ "$VERBOSE" == "true" ]] && echo "Validating $file..."
    
    # First validate JSON syntax
    if ! jq empty "$file" 2>/dev/null; then
        echo "::error file=$file::Invalid JSON syntax"
        has_any_errors=true
        [[ "$VERBOSE" == "true" ]] && echo "❌ $file: Invalid JSON syntax"
        continue
    fi
    
    # Then validate structure
    if validate_translation "$file"; then
        valid_files=$((valid_files + 1))
    else
        has_any_errors=true
    fi
done

# Summary
if [[ "$VERBOSE" == "true" ]]; then
    echo ""
    echo "================================"
    echo "Validation Summary:"
    echo "  Total files: $total_files"
    echo "  Valid files: $valid_files"
    echo "  Files with issues: $((total_files - valid_files))"
    
    if [[ "$has_any_errors" == "true" ]]; then
        echo "  Status: ❌ FAILED"
    else
        echo "  Status: ✅ PASSED"
    fi
elif [[ "$has_any_errors" == "true" ]]; then
    echo ""
    echo "Translation validation failed: $((total_files - valid_files))/$total_files files have issues"
fi

# Exit with error code if any validation failed
if [[ "$has_any_errors" == "true" ]]; then
    exit 1
fi

[[ "$VERBOSE" == "true" ]] && echo "All translation files are valid!"