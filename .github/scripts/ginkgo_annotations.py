import os
import re
import sys

if len(sys.argv) < 2:
    print("Usage: ginkgo_annotations.py <test log>")
    sys.exit(1)

log_file = sys.argv[1]

failures = []

with open(log_file, 'r') as f:
    lines = f.readlines()

for i, line in enumerate(lines):
    m = re.match(r"\s*\[FAIL\] (.*)", line)
    if m and i + 1 < len(lines):
        name = m.group(1).strip()
        path_line = lines[i+1]
        m2 = re.search(r"(/.*\.go):(\d+)", path_line)
        if m2:
            path = m2.group(1)
            line_no = m2.group(2)
            # convert path to repo-relative if running in GitHub Actions
            repo = os.getenv('GITHUB_WORKSPACE', '')
            if repo and path.startswith(repo):
                path = os.path.relpath(path, repo)
            failures.append((path, line_no, name))

for path, line_no, name in failures:
    # Generate GitHub Actions error annotation
    print(f"::error file={path},line={line_no},title=Test Failure::{name}")
