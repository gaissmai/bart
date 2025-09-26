#!/bin/bash
# Generate monomorphized methods for all node types from template files.

set -euo pipefail

# Use GOFILE environment variable set by go generate
template_file="${GOFILE}"

# Check if template exists
if [[ ! -f "${template_file}" ]]; then
    echo "Error: Template file '$template_file' not found" >&2
    echo "GOFILE=${GOFILE:-<not set>}" >&2
    exit 1
fi

echo "START: Generating monomorphized node methods from template '${GOFILE}' ..."

if grep -q "TODO" "${template_file}"; then
	echo "✗ Aborting, found pattern 'TODO' in template" >&2
	exit 1
fi

# Node types to generate
readonly NODE_TYPES=("bartNode" "fastNode" "liteNode")

# for goimports, see below
generated_files=()

for nodeType in "${NODE_TYPES[@]}"; do
    # build output filename e.g. bartiterators_gen.go
    output_file="${nodeType,,}"                             # lowercase e.g. bartNode -> bartnode
    output_file="${output_file}${template_file/_tmpl/_gen}" # concat with mangled template filename
    output_file="${output_file//node/}"                     # remove node in filename
    
    # Remove go:generate directives and build constraint, add generated header, substitute node type
    sed -e '/^\/\/go:generate\b/d' \
        -e '/Usage:.*go generate\b/d' \
        -e "s|^//go:build ignore.*$|// Code generated from file \"${template_file}\"; DO NOT EDIT.|" \
        -e '/GENERATE DELETE START/,/GENERATE DELETE END/d' \
        -e "s|_NODE_TYPE|${nodeType}|g" \
        "${template_file}" > "${output_file}"
    
    if [[ -f "${output_file}" ]]; then
        echo "✓ Generated ${output_file}"
        generated_files+=("$output_file")
    else
        echo "✗ Failed to generate ${output_file}" >&2
        exit 1
    fi
done

echo

# Run goimports on generated files
if command -v goimports >/dev/null 2>&1; then
    echo "Running goimports on generated files..."
    goimports -w "${generated_files[@]}"
    echo "✓ goimports completed"
else
    echo "⚠ goimports not found, skipping imports"
fi

# Run gofumpt on generated files
if command -v gofumpt >/dev/null 2>&1; then
    echo "Running gofumpt on generated files..."
    gofumpt -w "${generated_files[@]}"
    echo "✓ gofumpt completed"
else
    echo "⚠ gofumpt not found, skipping formatting"
fi

echo "END: Template generation complete!"
echo
