#!/bin/sh

set -e

I18N_DIR=resources/i18n

# Function to process JSON: remove empty attributes and sort
process_json() {
  jq 'walk(if type == "object" then with_entries(select(.value != null and .value != "" and .value != [] and .value != {})) | to_entries | sort_by(.key) | from_entries else . end)' "$1"
}

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

for file in ${I18N_DIR}/*.json; do
  name=$(basename "$file")
  code=$(echo "$name" | cut -f1 -d.)
  lang=$(jq -r .languageName < "$file")
  echo "Downloading $lang ($code)"
  check_lang_diff "$code"
done


# List changed languages to stderr
languages=""
for file in $(git diff --name-only --exit-code | grep json); do
  lang=$(jq -r .languageName < "$file")
  languages="${languages}$(echo $lang | tr -d '\n'), "
done
echo "${languages%??}" 1>&2