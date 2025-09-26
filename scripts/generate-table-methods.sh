#!/bin/bash
# Generate monomorphized methods for all table types from template files.

set -euo pipefail

function INFO  {                    echo -e "\e[34mINFO:\e[0m $1" ;         }
function WARN  {                    echo -e "\e[31mWARN:\e[0m $1" ;         }
function DIE   {                    echo -e "\e[31mERROR:\e[0m $1"; exit 1; }

# Use GOFILE environment variable set by go generate
template_file="${GOFILE}"

# Check if template exists
if [[ ! -f "${template_file}" ]]; then
    WARN "Error: Template file '$template_file' not found" >&2
    DIE  "GOFILE=${GOFILE:-<not set>}" >&2
fi

INFO "START: Generating monomorphized table methods from template '${GOFILE}' ..."

if grep -q "TODO" "${template_file}"; then
    DIE "✗ Aborting, found pattern 'TODO' in ${template_file}" >&2
fi

# Node types to generate
readonly TABLE_TYPES=("Table" "Fast" "liteTable")

# for goimports, see below
generated_files=()

for tableType in "${TABLE_TYPES[@]}"; do
    # build output filename e.g. bartiterators_gen.go
    output_file="${tableType,,}"                            # lowercase e.g. Fast -> fast
    output_file="${output_file}${template_file/_tmpl/_gen}" # concat with mangled template filename
    output_file="${output_file/#table/bart}"       # TODO
    output_file="${output_file/#litetable/lite}"       # TODO

    # Remove go:generate directives and build constraint, add generated header, substitute node type
    sed -e '/^\/\/go:generate\b/d' \
        -e '/Usage:.*go generate\b/d' \
        -e "s|^//go:build ignore.*$|// Code generated from file \"${template_file}\"; DO NOT EDIT.|" \
        -e '/GENERATE DELETE START/,/GENERATE DELETE END/d' \
        -e "s|_TABLE_TYPE|${tableType}|g" \
        "${template_file}" > "${output_file}"

    if [[ -f "${output_file}" ]]; then
        INFO "✓ Generated ${output_file}"
        generated_files+=("$output_file")
    else
        DIE "✗ Failed to generate ${output_file}" >&2
    fi
done

echo

# Run goimports on generated files
if command -v goimports >/dev/null 2>&1; then
    INFO "Running goimports on generated files..."
    goimports -w "${generated_files[@]}"
    INFO "✓ goimports completed"
else
    WARN "⚠ goimports not found, skipping imports"
fi

# Run gofumpt on generated files
if command -v gofumpt >/dev/null 2>&1; then
    INFO "Running gofumpt on generated files..."
    gofumpt -w "${generated_files[@]}"
    INFO "✓ gofumpt completed"
else
    WARN "⚠ gofumpt not found, skipping formatting"
fi

INFO "END: Template generation complete!"
echo
