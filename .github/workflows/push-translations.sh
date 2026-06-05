#!/bin/sh

set -e

I18N_DIR=resources/i18n

# Normalize JSON for deterministic comparison:
# remove empty/null attributes, sort keys alphabetically
process_json() {
  jq 'walk(if type == "object" then with_entries(select(.value != null and .value != "" and .value != [] and .value != {})) | to_entries | sort_by(.key) | from_entries else . end)' "$1"
}

# Get list of all languages configured in the POEditor project
get_language_list() {
  curl -s -X POST https://api.poeditor.com/v2/languages/list \
    -d api_token="${POEDITOR_APIKEY}" \
    -d id="${POEDITOR_PROJECTID}"
}

# Extract language name from the language list JSON given a language code
get_language_name() {
  lang_code="$1"
  lang_list="$2"
  echo "$lang_list" | jq -r ".result.languages[] | select(.code == \"$lang_code\") | .name"
}

# Extract language code from a file path (e.g., "resources/i18n/fr.json" -> "fr")
get_lang_code() {
  filepath="$1"
  filename=$(basename "$filepath")
  echo "${filename%.*}"
}

# Export the current translation for a language from POEditor (v2 API)
export_language() {
  lang_code="$1"
  response=$(curl -s -X POST https://api.poeditor.com/v2/projects/export \
    -d api_token="${POEDITOR_APIKEY}" \
    -d id="${POEDITOR_PROJECTID}" \
    -d language="$lang_code" \
    -d type="key_value_json")

  url=$(echo "$response" | jq -r '.result.url')
  if [ -z "$url" ] || [ "$url" = "null" ]; then
    echo "Failed to export $lang_code: $response" >&2
    return 1
  fi
  echo "$url"
}

# Flatten nested JSON to POEditor languages/update format.
# POEditor uses term + context pairs, where:
#   term = the leaf key name
#   context = the parent path as "key1"."key2"."key3" (empty for root keys)
flatten_to_poeditor() {
  jq -c '[paths(scalars) as $p |
    {
      "term": ($p | last | tostring),
      "context": (if ($p | length) > 1 then ($p[:-1] | map("\"" + tostring + "\"") | join(".")) else "" end),
      "translation": {"content": getpath($p)}
    }
  ]' "$1"
}

# Update translations for a language in POEditor via languages/update API
update_language() {
  lang_code="$1"
  file="$2"

  flatten_to_poeditor "$file" > /tmp/poeditor_data.json
  response=$(curl -s -X POST https://api.poeditor.com/v2/languages/update \
    -d api_token="${POEDITOR_APIKEY}" \
    -d id="${POEDITOR_PROJECTID}" \
    -d language="$lang_code" \
    --data-urlencode data@/tmp/poeditor_data.json)
  rm -f /tmp/poeditor_data.json

  status=$(echo "$response" | jq -r '.response.status')
  if [ "$status" != "success" ]; then
    echo "Failed to update $lang_code: $response" >&2
    return 1
  fi

  parsed=$(echo "$response" | jq -r '.result.translations.parsed')
  added=$(echo "$response" | jq -r '.result.translations.added')
  updated=$(echo "$response" | jq -r '.result.translations.updated')
  echo "  Translations - parsed: $parsed, added: $added, updated: $updated"
}

# --- Main ---

if [ $# -eq 0 ]; then
  echo "Usage: $0 <file1> [file2] ..."
  echo "No files specified. Nothing to do."
  exit 0
fi

lang_list=$(get_language_list)
upload_count=0

for file in "$@"; do
  if [ ! -f "$file" ]; then
    echo "Warning: File not found: $file, skipping"
    continue
  fi

  lang_code=$(get_lang_code "$file")
  lang_name=$(get_language_name "$lang_code" "$lang_list")

  if [ -z "$lang_name" ]; then
    echo "Warning: Language code '$lang_code' not found in POEditor, skipping $file"
    continue
  fi

  echo "Processing $lang_name ($lang_code)..."

  # Export current state from POEditor
  url=$(export_language "$lang_code")
  curl -sSL "$url" -o poeditor_export.json

  # Normalize both files for comparison
  process_json "$file" > local_normalized.json
  process_json poeditor_export.json > remote_normalized.json

  # Compare normalized versions
  if diff -q local_normalized.json remote_normalized.json > /dev/null 2>&1; then
    echo "  No differences, skipping"
  else
    echo "  Differences found, updating POEditor..."
    update_language "$lang_code" "$file"
    upload_count=$((upload_count + 1))
  fi

  rm -f poeditor_export.json local_normalized.json remote_normalized.json
done

echo ""
echo "Done. Updated $upload_count translation(s) in POEditor."
