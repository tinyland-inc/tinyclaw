#!/usr/bin/env python3
"""Config drift detection for PicoClaw.

Compares the running PicoClaw config against the Dhall source of truth.
Reports any divergence between the rendered Dhall config and the active
JSON config.

Usage:
    python3 scripts/drift-check.py [--config PATH] [--dhall PATH] [--json]

Exit codes:
    0 - No drift detected
    1 - Drift detected
    2 - Error (missing files, invalid config, etc.)
"""

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path


def load_json(path: str) -> dict:
    """Load and parse a JSON file."""
    with open(path) as f:
        return json.load(f)


def render_dhall(path: str) -> dict:
    """Render a Dhall file to JSON via dhall-to-json."""
    try:
        result = subprocess.run(
            ["dhall-to-json", "--file", path],
            capture_output=True,
            text=True,
            timeout=30,
        )
        if result.returncode != 0:
            print(f"Error: dhall-to-json failed: {result.stderr}", file=sys.stderr)
            sys.exit(2)
        return json.loads(result.stdout)
    except FileNotFoundError:
        print("Error: dhall-to-json not found in PATH", file=sys.stderr)
        sys.exit(2)
    except subprocess.TimeoutExpired:
        print("Error: dhall-to-json timed out", file=sys.stderr)
        sys.exit(2)


def diff_dicts(a: dict, b: dict, path: str = "") -> list[str]:
    """Recursively compare two dicts and return list of differences."""
    diffs = []
    all_keys = set(list(a.keys()) + list(b.keys()))

    for key in sorted(all_keys):
        current_path = f"{path}.{key}" if path else key
        a_val = a.get(key)
        b_val = b.get(key)

        if key not in a:
            diffs.append(f"  + {current_path}: {json.dumps(b_val)}")
        elif key not in b:
            diffs.append(f"  - {current_path}: {json.dumps(a_val)}")
        elif isinstance(a_val, dict) and isinstance(b_val, dict):
            diffs.extend(diff_dicts(a_val, b_val, current_path))
        elif isinstance(a_val, list) and isinstance(b_val, list):
            if json.dumps(a_val, sort_keys=True) != json.dumps(b_val, sort_keys=True):
                diffs.append(f"  ~ {current_path}: list differs")
        elif a_val != b_val:
            # Skip credential fields (they're redacted in Dhall)
            if any(cred in key for cred in ["api_key", "token", "secret", "auth_key"]):
                continue
            diffs.append(f"  ~ {current_path}: {json.dumps(a_val)} -> {json.dumps(b_val)}")

    return diffs


def main():
    parser = argparse.ArgumentParser(description="PicoClaw config drift detection")
    parser.add_argument(
        "--config",
        default=os.path.expanduser("~/.picoclaw/config.json"),
        help="Path to active JSON config",
    )
    parser.add_argument(
        "--dhall",
        default=os.path.expanduser("~/.picoclaw/config.dhall"),
        help="Path to Dhall source of truth",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output diff as JSON",
    )
    args = parser.parse_args()

    # Check files exist
    if not Path(args.config).exists():
        print(f"Error: config file not found: {args.config}", file=sys.stderr)
        sys.exit(2)

    if not Path(args.dhall).exists():
        print(f"Error: dhall file not found: {args.dhall}", file=sys.stderr)
        sys.exit(2)

    # Load both configs
    active_config = load_json(args.config)
    dhall_config = render_dhall(args.dhall)

    # Compare
    diffs = diff_dicts(dhall_config, active_config)

    if not diffs:
        print("No drift detected")
        sys.exit(0)

    if args.json:
        print(json.dumps({"drift_count": len(diffs), "diffs": diffs}, indent=2))
    else:
        print(f"Drift detected ({len(diffs)} differences):")
        for diff in diffs:
            print(diff)

    sys.exit(1)


if __name__ == "__main__":
    main()
