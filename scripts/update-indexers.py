#!/usr/bin/env python3

import os
import re
from typing import Dict

# Paths relative to GitHub Actions workspace
DEFINITIONS_DIR = "../autobrr/internal/indexer/definitions"
INDEXERS_OUTPUT = "../autobrr.com/snippets/indexers.mdx"
FREELEECH_OUTPUT = "../autobrr.com/snippets/freeleech.mdx"

def parse_yaml_file(file_path: str) -> Dict:
    """Parse YAML files"""
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
        
        # Look for freeleech in vars section
        has_freeleech = bool(re.search(r'vars:.*?-\s*freeleech\s*$', content, re.MULTILINE | re.DOTALL))
        has_freeleech_percent = bool(re.search(r'vars:.*?-\s*freeleechPercent\s*$', content, re.MULTILINE | re.DOTALL))
        
        result = {
            'name': name_match.group(1).strip() if name_match else '',
            'description': desc_match.group(1).strip() if desc_match else '',
            'supports': supports,
            'freeleech': has_freeleech,
            'freeleechPercent': has_freeleech_percent
        }
    return result

def get_feature_checkmark(value: bool) -> str:
    return "✓" if value else "✗"

def generate_indexers_markdown(indexers: list) -> str:
    """Generate markdown for indexers table."""
    markdown = "<details>\n\n"
    markdown += "<summary>Click to view supported indexers</summary>\n\n"
    markdown += "| Indexer | Description | IRC | RSS |\n"
    markdown += "|---------|-------------|-----|-----|\n"
    
    for indexer in indexers:
        name = indexer.get('name', '')
        description = indexer.get('description', '')
        irc_support = get_feature_checkmark('irc' in [f.lower() for f in indexer.get('supports', [])])
        rss_support = get_feature_checkmark('rss' in [f.lower() for f in indexer.get('supports', [])])
        
        markdown += f"| {name} | {description} | {irc_support} | {rss_support} |\n"
    
    markdown += "\n</details>"
    return markdown

def generate_freeleech_markdown(indexers: list) -> str:
    """Generate markdown for freeleech table."""
    markdown = "| Indexer | Freeleech | Freeleech Percent |\n"
    markdown += "|---------|-----------|------------------|\n"
    
    for indexer in indexers:
        # Skip if neither freeleech feature is supported
        if not (indexer.get('freeleech', False) or indexer.get('freeleechPercent', False)):
            continue
            
        name = indexer.get('name', '')
        freeleech = get_feature_checkmark(indexer.get('freeleech', False))
        freeleech_percent = get_feature_checkmark(indexer.get('freeleechPercent', False))
        
        markdown += f"| {name} | {freeleech} | {freeleech_percent} |\n"
    
    return markdown

def main():
    """Generate markdown documents"""
    indexers = []
    
    # Read the YAML files
    for filename in os.listdir(DEFINITIONS_DIR):
        if filename.endswith('.yaml'):
            file_path = os.path.join(DEFINITIONS_DIR, filename)
            indexer = parse_yaml_file(file_path)
            if indexer:
                indexers.append(indexer)
    
    # Sort indexers by name, but put generic ones last
    def sort_key(indexer):
        name = indexer.get('name', '').lower()
        return (name.startswith('generic'), name)
    
    indexers.sort(key=sort_key)
    
    # Generate and write files
    for output_file, content in [
        (INDEXERS_OUTPUT, generate_indexers_markdown(indexers)),
        (FREELEECH_OUTPUT, generate_freeleech_markdown(indexers))
    ]:
        os.makedirs(os.path.dirname(output_file), exist_ok=True)
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(content)

if __name__ == "__main__":
    main()
