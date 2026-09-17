"""Check that each workflow's `result` job waits on every other job.

A workflow's `result` job is the one check the branch ruleset requires for it.
It only fails when a job it lists in `needs` fails, so a job left off that list
can fail without blocking a merge. Run on workflow files; exits non-zero if any
`result` job misses a job or names one that doesn't exist.
"""

import sys

import yaml


def check(path: str) -> list[str]:
    with open(path, encoding="utf-8") as f:
        jobs = (yaml.safe_load(f) or {}).get("jobs") or {}
    if "result" not in jobs:
        return []

    needs = jobs["result"].get("needs") or []
    if isinstance(needs, str):
        needs = [needs]

    problems = []
    missing = sorted(set(jobs) - {"result"} - set(needs))
    if missing:
        problems.append(f"{path}: `result` does not wait on: {', '.join(missing)}")
    unknown = sorted(set(needs) - set(jobs))
    if unknown:
        problems.append(f"{path}: `result` waits on jobs that don't exist: {', '.join(unknown)}")
    return problems


def main(paths: list[str]) -> int:
    problems = [p for path in paths for p in check(path)]
    for problem in problems:
        print(problem)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
