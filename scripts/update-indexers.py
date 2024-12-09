#!/usr/bin/env python3

import os
import re
from typing import Dict

# Paths relative to GitHub Actions workspace
DEFINITIONS_DIR = "autobrr/internal/indexer/definitions"
OUTPUT_FILE = "autobrr.com/snippets/indexers.mdx"

def parse_yaml_file(file_path: str) -> Dict:
    """Parse YAML file manually for the fields we need."""
    result = {}
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
        
        # Extract fields using regex
        name_match = re.search(r'name:\s*(.*)', content)
        desc_match = re.search(r'description:\s*(.*)', content)
        
        supports_match = re.search(r'supports:\s*\n((?:\s*-\s*[^\n]+\n)*)', content)
        supports = []
        if supports_match:
            supports = re.findall(r'-\s*(\w+)', supports_match.group(1))
        
        result = {
            'name': name_match.group(1).strip() if name_match else '',
            'description': desc_match.group(1).strip() if desc_match else '',
            'supports': supports
        }
    return result

def get_feature_checkmark(indexer: Dict, feature: str) -> str:
    """Return checkmark or cross based on feature support."""
    return "✓" if feature.lower() in [f.lower() for f in indexer.get('supports', [])] else "✗"

def generate_markdown(definitions_dir: str, output_file: str):
    """Generate markdown document for all indexers."""
    indexers = []
    
    # Read the YAML files
    for filename in os.listdir(definitions_dir):
        if filename.endswith('.yaml'):
            file_path = os.path.join(definitions_dir, filename)
            indexer = parse_yaml_file(file_path)
            if indexer:
                indexers.append(indexer)
    
    # Sort indexers by name, but put generic ones last
    def sort_key(indexer):
        name = indexer.get('name', '').lower()
        return (name.startswith('generic'), name)
    
    indexers.sort(key=sort_key)
    
    # Generate markdown content
    markdown = "<details>\n\n"
    markdown += "<summary>Click to view supported indexers</summary>\n\n"
    markdown += "| Indexer | Description | IRC | RSS |\n"
    markdown += "|---------|-------------|-----|-----|\n"
    
    last_was_generic = False
    for indexer in indexers:
        name = indexer.get('name', '').lower()
        
        if name.startswith('generic') and not last_was_generic:
            last_was_generic = True
            
        name = indexer.get('name', '')
        description = indexer.get('description', '')
        irc_support = get_feature_checkmark(indexer, 'irc')
        rss_support = get_feature_checkmark(indexer, 'rss')
        
        markdown += f"| {name} | {description} | {irc_support} | {rss_support} |\n"
    
    markdown += "\n</details>"

    # Ensure output directory exists
    os.makedirs(os.path.dirname(output_file), exist_ok=True)
    
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(markdown)

if __name__ == "__main__":
    generate_markdown(DEFINITIONS_DIR, OUTPUT_FILE)