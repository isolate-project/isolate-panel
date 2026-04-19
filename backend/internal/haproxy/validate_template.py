#!/usr/bin/env python3
"""
Simple HAProxy template syntax validator
Checks for balanced braces and basic template syntax
"""

import re
import sys

def validate_template_syntax(content):
    """Validate Go template syntax"""
    errors = []

    brace_stack = []
    in_template = False
    template_start = 0
    line_num = 1
    col_num = 1

    for i, char in enumerate(content):
        if char == '\n':
            line_num += 1
            col_num = 1
        else:
            col_num += 1

        if char == '{':
            if i + 1 < len(content) and content[i + 1] == '{':
                in_template = True
                template_start = i
                brace_stack.append(('{{', line_num, col_num))
        elif char == '}':
            if i + 1 < len(content) and content[i + 1] == '}':
                if not brace_stack:
                    errors.append(f"Line {line_num}, Col {col_num}: Unmatched closing '}}'")
                elif brace_stack[-1][0] != '{{':
                    errors.append(f"Line {line_num}, Col {col_num}: Mismatched braces")
                else:
                    brace_stack.pop()

    for brace, line, col in brace_stack:
        errors.append(f"Line {line}, Col {col}: Unclosed '{brace}'")

    required_vars = [
        r'\{\{range \$port, \$group := \.PortGroups\}\}',
        r'\{\{range \.Backends\}\}',
        r'\{\{\.StatsPassword\}\}',
    ]

    for var_pattern in required_vars:
        if not re.search(var_pattern, content):
            errors.append(f"Missing required template variable pattern: {var_pattern}")

    required_sections = [
        'global',
        'defaults',
        'frontend',
        'backend',
        'stats socket',
        'maxconn',
        'mode tcp',
    ]

    for section in required_sections:
        if section not in content:
            errors.append(f"Missing required HAProxy section: {section}")

    return errors

def main():
    if len(sys.argv) != 2:
        print("Usage: python3 validate_template.py <template_file>")
        sys.exit(1)

    template_file = sys.argv[1]

    try:
        with open(template_file, 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print(f"Error: Template file not found: {template_file}")
        sys.exit(1)

    errors = validate_template_syntax(content)

    if errors:
        print(f"❌ Template validation failed with {len(errors)} error(s):")
        for error in errors:
            print(f"  - {error}")
        sys.exit(1)
    else:
        print("✅ Template syntax validation passed!")
        print(f"   Template file: {template_file}")
        print(f"   Total length: {len(content)} characters")
        print(f"   Template variables found: {content.count('{{')}")
        sys.exit(0)

if __name__ == '__main__':
    main()