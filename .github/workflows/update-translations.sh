#!/bin/sh

set -e

I18N_DIR=resources/i18n

# Function to process JSON: remove empty attributes and sort
process_json() {
  jq 'walk(if type == "object" then with_entries(select(.value != null and .value != "" and .value != [] and .value != {})) | to_entries | sort_by(.key) | from_entries else . end)' "$1"
}

# Function to check differences between local and remote translations
check_lang_diff() {
  filename=${I18N_DIR}/"$1".json
  url=$(curl -s -X POST https://poeditor.com/api/ \
    -d api_token="${POEDITOR_APIKEY}" \
    -d action="export" \
    -d id="${POEDITOR_PROJECTID}" \
    -d language="$1" \
    -d type="key_value_json" | jq -r .item)
  if [ -z "$url" ]; then
    echo "Failed to export $1"
    return 1
  fi
  curl -sSL "$url" > poeditor.json

  process_json "$filename" > "$filename".tmp
  process_json poeditor.json > poeditor.tmp
  
  diff=$(diff -u "$filename".tmp poeditor.tmp) || true
  if [ -n "$diff" ]; then
    echo "$diff"
    mv poeditor.json "$filename"
  fi
  
  rm -f poeditor.json poeditor.tmp "$filename".tmp
}

# Function to get the list of languages
get_language_list() {
  response=$(curl -s -X POST https://api.poeditor.com/v2/languages/list \
    -d api_token="${POEDITOR_APIKEY}" \
    -d id="${POEDITOR_PROJECTID}")

  echo $response
}

# Function to get the language name from the language code
get_language_name() {
  lang_code="$1"
  lang_list="$2"

  lang_name=$(echo "$lang_list" | jq -r ".result.languages[] | select(.code == \"$lang_code\") | .name")

  if [ -z "$lang_name" ]; then
    echo "Error: Language code '$lang_code' not found" >&2
    return 1
  fi

  echo "$lang_name"
}

# Function to get the language code from the file path
get_lang_code() {
  filepath="$1"
  # Extract just the filename
  filename=$(basename "$filepath")

  # Remove the extension
  lang_code="${filename%.*}"

  echo "$lang_code"
}

lang_list=$(get_language_list)

# Check differences for each language
for file in ${I18N_DIR}/*.json; do
  code=$(get_lang_code "$file")
  lang=$(jq -r .languageName < "$file")
  lang_name=$(get_language_name "$code" "$lang_list")
  echo "Downloading $lang_name - $lang ($code)"
  check_lang_diff "$code"
done

# List changed languages to stderr
languages=""
for file in $(git diff --name-only --exit-code | grep json); do
  lang_code=$(get_lang_code "$file")
  lang_name=$(get_language_name "$lang_code" "$lang_list")
  languages="${languages}$(echo "$lang_name" | tr -d '\n'), "
done
echo "${languages%??}" 1>&2