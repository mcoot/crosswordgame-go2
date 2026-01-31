#!/usr/bin/env python3
"""Filter words.txt to common English words using wordfreq Zipf frequency scores.

Usage:
    python scripts/filter_wordlist.py analyze          # Show distribution and samples
    python scripts/filter_wordlist.py filter [THRESHOLD] # Filter words.txt (default threshold: 3.0)
"""

import sys
from pathlib import Path
from collections import Counter

from wordfreq import zipf_frequency

WORDS_FILE = Path(__file__).parent.parent / "data" / "words.txt"
MIN_LENGTH = 2


def load_words() -> list[str]:
    return [line.strip().lower() for line in WORDS_FILE.read_text().splitlines() if line.strip()]


def get_scores(words: list[str]) -> list[tuple[str, float]]:
    return [(w, zipf_frequency(w, "en")) for w in words]


def analyze(words: list[str]) -> None:
    scores = get_scores(words)
    total = len(scores)

    # Bucket into Zipf ranges
    buckets: dict[str, list[str]] = {
        "0 (not found)": [],
        "0.01-1.0": [],
        "1.0-2.0": [],
        "2.0-2.5": [],
        "2.5-3.0": [],
        "3.0-3.5": [],
        "3.5-4.0": [],
        "4.0-5.0": [],
        "5.0+": [],
    }

    for word, score in scores:
        if score == 0:
            buckets["0 (not found)"].append(word)
        elif score < 1.0:
            buckets["0.01-1.0"].append(word)
        elif score < 2.0:
            buckets["1.0-2.0"].append(word)
        elif score < 2.5:
            buckets["2.0-2.5"].append(word)
        elif score < 3.0:
            buckets["2.5-3.0"].append(word)
        elif score < 3.5:
            buckets["3.0-3.5"].append(word)
        elif score < 4.0:
            buckets["3.5-4.0"].append(word)
        elif score < 5.0:
            buckets["4.0-5.0"].append(word)
        else:
            buckets["5.0+"].append(word)

    print(f"Total words: {total}")
    print(f"\n{'Zipf Range':<20} {'Count':>8} {'%':>7}  Sample Words")
    print("-" * 90)

    for label, bucket_words in buckets.items():
        count = len(bucket_words)
        pct = count / total * 100
        sample = ", ".join(sorted(bucket_words)[:8])
        print(f"{label:<20} {count:>8} {pct:>6.1f}%  {sample}")

    # Show cumulative counts at various thresholds
    print(f"\n{'Threshold (>=)':<20} {'Words Kept':>12} {'% of Original':>15}")
    print("-" * 50)
    for threshold in [0.01, 1.0, 2.0, 2.5, 3.0, 3.5, 4.0]:
        kept = sum(1 for _, s in scores if s >= threshold and len(_) >= MIN_LENGTH)
        print(f"Zipf >= {threshold:<12} {kept:>12} {kept / total * 100:>14.1f}%")


def filter_words(words: list[str], threshold: float) -> list[str]:
    return [
        w
        for w, score in get_scores(words)
        if score >= threshold and len(w) >= MIN_LENGTH
    ]


def main() -> None:
    if len(sys.argv) < 2 or sys.argv[1] not in ("analyze", "filter"):
        print(__doc__)
        sys.exit(1)

    words = load_words()
    command = sys.argv[1]

    if command == "analyze":
        analyze(words)
    elif command == "filter":
        threshold = float(sys.argv[2]) if len(sys.argv) > 2 else 3.0
        filtered = filter_words(words, threshold)
        filtered.sort()
        print(f"Filtering with Zipf >= {threshold}, min length >= {MIN_LENGTH}")
        print(f"Input: {len(words)} words")
        print(f"Output: {len(filtered)} words")

        WORDS_FILE.write_text("\n".join(filtered) + "\n")
        print(f"Written to {WORDS_FILE}")


if __name__ == "__main__":
    main()
