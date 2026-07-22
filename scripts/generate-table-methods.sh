#!/bin/bash
# Generate monomorphized methods for all table types from template files.

set -euo pipefail

function INFO  { echo -e "\e[34mINFO:\e[0m $1" ; }
function WARN  { echo -e "\e[31mWARN:\e[0m $1" ; }
function DIE   { echo -e "\e[31mERROR:\e[0m $1"; exit 1; }

# Use GOFILE environment variable set by go generate
template_file="${GOFILE:-}"

# Check if template exists
if [[ -z "${template_file}" || ! -f "${template_file}" ]]; then
    WARN "Error: Template file '$template_file' not found" >&2
    DIE  "GOFILE=${GOFILE:-<not set>}" >&2
fi

INFO "START: Generating monomorphized table methods from template '${template_file}' ..."

if grep -q "TODO" "${template_file}"; then
    DIE "✗ Aborting, found pattern 'TODO' in ${template_file}" >&2
fi

# Mapping: TABLE_TYPE -> NODE_TYPE
declare -A NODE_TYPES=(
    ["Table"]="BartNode"
    ["Fast"]="FastNode"
    ["liteTable"]="LiteNode"
)

generated_files=()

for tableType in "${!NODE_TYPES[@]}"; do
    nodeType="${NODE_TYPES[$tableType]}"

    # Determine output filename prefix
    type_prefix="${tableType,,}"                    # lowercase
    type_prefix="${type_prefix/#table/bart}"        # table     -> bart
    type_prefix="${type_prefix/#litetable/lite}"    # litetable -> lite

    template_base="${template_file##*common}"        # -> methods_tmpl.go
    base_mangled="${template_base/_tmpl/generated}"  # -> methodsgenerated.go
    output_file="${type_prefix}${base_mangled}"      # e.g. -> bartmethodsgenerated.go

    # Use an array to pass flags and expressions safely to sed without quote-splitting issues
    lite_filter=()
    if [[ "${nodeType,,}" == *lite* ]]; then
        # Delete all lines between the markers (inclusive) for LiteNode
        lite_filter+=(-e '/GENERATE SKIP_LITE START/,/GENERATE SKIP_LITE END/d')
    else
        # For Table/Fast nodes, keep the enclosed code and remove only the marker comment lines
        lite_filter+=(-e '/GENERATE SKIP_LITE/d')
    fi

    # Single-pass sed substitution:
    # 1. Insert generated header
    # 2. Strip build tags and go:generate directives
    # 3. Strip local dev stub block between GENERATE DELETE markers
    # 4. Replace _TABLE_TYPE and _NODE_TYPE placeholders
    sed -e "1i\\
// Code generated from file \"${template_file}\"; DO NOT EDIT." \
        -e '/^\/\/go:build generate\b/d' \
        -e '/^\/\/go:generate\b/d' \
        -e '/GENERATE DELETE START/,/GENERATE DELETE END/d' \
        "${lite_filter[@]}" \
        -e "s|_TABLE_TYPE|${tableType}|g" \
        -e "s|_NODE_TYPE|${nodeType}|g" \
        "${template_file}" > "${output_file}"

    if [[ -f "${output_file}" ]]; then
        INFO "✓ Generated ${output_file} (${tableType} -> ${nodeType})"
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
