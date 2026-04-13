#!/usr/bin/env python3

import json
import os
import re
import subprocess
import sys
import time


def sanitize_output(text: str) -> str:
    replacements = [
        (
            r"/var/folders/[^\s'\"]+/tmp\.[A-Za-z0-9]+",
            "/workspace/gig/demo-run",
        ),
        (
            r"/tmp/tmp\.[A-Za-z0-9]+",
            "/workspace/gig/demo-run",
        ),
    ]
    for pattern, replacement in replacements:
        text = re.sub(pattern, replacement, text)
    return text


def main() -> int:
    if len(sys.argv) < 3:
        print("usage: build_cast.py <output-path> <command> [args...]", file=sys.stderr)
        return 2

    output_path = sys.argv[1]
    command = sys.argv[2:]

    proc = subprocess.run(command, capture_output=True, text=True, check=True)
    text = sanitize_output(proc.stdout)

    lines = text.splitlines(keepends=True)
    width = max(90, min(140, max((len(line.rstrip("\n")) for line in lines), default=90)))
    height = 34

    header = {
        "version": 2,
        "width": width,
        "height": height,
        "timestamp": int(time.time()),
        "env": {
            "TERM": "xterm-256color",
            "SHELL": "/bin/bash",
        },
    }

    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    current_time = 0.0
    with open(output_path, "w", encoding="utf-8") as handle:
        handle.write(json.dumps(header) + "\n")
        for raw_line in lines:
            line = raw_line.rstrip("\n")
            event_text = line + "\r\n"
            handle.write(json.dumps([round(current_time, 2), "o", event_text]) + "\n")
            if line.startswith("$ "):
                current_time += 0.7
            elif line.strip() == "":
                current_time += 0.12
            else:
                current_time += 0.08

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
