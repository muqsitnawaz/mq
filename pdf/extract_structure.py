#!/usr/bin/env python3
"""
Extract PDF structure using PyMuPDF (fitz).

Outputs JSON with:
- title: inferred document title
- headings: list of {level, text, page, font_size}
- tables: list of detected tables (basic detection)

Headings are inferred from font size relative to body text.
"""

import json
import os
import sys
import warnings
from collections import Counter
from dataclasses import dataclass
from typing import Optional

# Suppress all warnings
warnings.filterwarnings("ignore")
os.environ["PYMUPDF_NOWARNING"] = "1"

import fitz  # PyMuPDF

# Suppress MuPDF warnings at runtime
fitz.TOOLS.mupdf_display_warnings(False)


@dataclass
class TextBlock:
    text: str
    font_size: float
    font_name: str
    is_bold: bool
    page: int
    x: float
    y: float


def extract_text_blocks(doc: fitz.Document) -> list[TextBlock]:
    """Extract all text blocks with font information."""
    blocks = []

    for page_num, page in enumerate(doc):
        # Get detailed text information
        text_dict = page.get_text("dict", flags=fitz.TEXT_PRESERVE_WHITESPACE)

        for block in text_dict.get("blocks", []):
            if block.get("type") != 0:  # Skip non-text blocks
                continue

            for line in block.get("lines", []):
                line_text = ""
                line_font_size = 0
                line_font_name = ""
                line_is_bold = False

                for span in line.get("spans", []):
                    text = span.get("text", "").strip()
                    if not text:
                        continue

                    line_text += text + " "
                    # Use the largest font in the line
                    font_size = span.get("size", 0)
                    if font_size > line_font_size:
                        line_font_size = font_size
                        line_font_name = span.get("font", "")
                        # Check for bold in font name or flags
                        flags = span.get("flags", 0)
                        line_is_bold = (
                            "bold" in line_font_name.lower() or
                            "black" in line_font_name.lower() or
                            (flags & 2 ** 4) != 0  # Bold flag
                        )

                line_text = line_text.strip()
                if line_text and line_font_size > 0:
                    bbox = line.get("bbox", [0, 0, 0, 0])
                    blocks.append(TextBlock(
                        text=line_text,
                        font_size=line_font_size,
                        font_name=line_font_name,
                        is_bold=line_is_bold,
                        page=page_num + 1,
                        x=bbox[0],
                        y=bbox[1],
                    ))

    return blocks


def find_body_font_size(blocks: list[TextBlock]) -> float:
    """Find the most common font size (body text)."""
    if not blocks:
        return 12.0

    # Round font sizes to avoid floating point issues
    sizes = [round(b.font_size, 1) for b in blocks]
    counter = Counter(sizes)

    # Most common size is likely body text
    most_common = counter.most_common(1)
    return most_common[0][0] if most_common else 12.0


def is_likely_heading(text: str) -> bool:
    """Heuristic to filter out non-heading text."""
    # Skip if too short or too long
    if len(text) < 2 or len(text) > 150:
        return False

    # Skip lines that look like body text (contain sentence punctuation mid-text)
    if ". " in text[:-1]:  # Period followed by space (not at end)
        return False

    # Skip lines that are mostly numbers/special chars (like arxiv IDs)
    alpha_count = sum(1 for c in text if c.isalpha())
    if alpha_count < len(text) * 0.5:
        return False

    # Skip lines starting with common non-heading patterns
    lower = text.lower()
    skip_patterns = [
        "provided ", "permission ", "reproduce ", "copyright ",
        "http://", "https://", "arxiv:", "doi:",
    ]
    for pattern in skip_patterns:
        if lower.startswith(pattern):
            return False

    # Skip author-like patterns (name followed by asterisk or affiliation number)
    if text.endswith("*") or text.endswith("\u2217"):
        return False

    return True


def infer_headings(blocks: list[TextBlock], body_size: float, min_ratio: float = 1.15) -> list[dict]:
    """Infer headings based on font size relative to body text."""
    headings = []
    seen_texts = set()  # Avoid duplicates

    for block in blocks:
        # Skip if already seen (exact match)
        if block.text in seen_texts:
            continue

        # Apply heading heuristics
        if not is_likely_heading(block.text):
            continue

        ratio = block.font_size / body_size if body_size > 0 else 1.0

        # Determine heading level based on font size ratio
        level = 0
        if ratio >= 2.0 or (ratio >= 1.5 and block.is_bold):
            level = 1
        elif ratio >= 1.5 or (ratio >= 1.3 and block.is_bold):
            level = 2
        elif ratio >= 1.25 or (ratio >= 1.15 and block.is_bold):
            level = 3
        elif ratio >= min_ratio or block.is_bold:
            level = 4

        if level > 0:
            headings.append({
                "level": level,
                "text": block.text,
                "page": block.page,
                "font_size": round(block.font_size, 2),
            })
            seen_texts.add(block.text)

    return headings


def detect_tables(doc: fitz.Document) -> list[dict]:
    """Basic table detection using PyMuPDF's table finder."""
    tables = []

    for page_num, page in enumerate(doc):
        try:
            # PyMuPDF 1.23+ has find_tables()
            page_tables = page.find_tables()
            for i, table in enumerate(page_tables):
                # Extract table data
                data = table.extract()
                if data and len(data) > 1:  # At least header + 1 row
                    tables.append({
                        "page": page_num + 1,
                        "rows": len(data),
                        "cols": len(data[0]) if data else 0,
                        "headers": data[0] if data else [],
                    })
        except AttributeError:
            # Older PyMuPDF version without find_tables
            pass

    return tables


def extract_structure(pdf_path: Optional[str] = None, pdf_bytes: Optional[bytes] = None) -> dict:
    """Extract structure from PDF file or bytes."""
    if pdf_bytes:
        doc = fitz.open(stream=pdf_bytes, filetype="pdf")
    elif pdf_path:
        doc = fitz.open(pdf_path)
    else:
        raise ValueError("Must provide pdf_path or pdf_bytes")

    try:
        # Extract text blocks with font info
        blocks = extract_text_blocks(doc)

        # Find body text size
        body_size = find_body_font_size(blocks)

        # Infer headings
        headings = infer_headings(blocks, body_size)

        # Detect tables
        tables = detect_tables(doc)

        # Infer title (first large heading, usually on page 1)
        title = ""
        for h in headings:
            if h["page"] == 1 and h["level"] <= 2:
                title = h["text"]
                break

        return {
            "title": title,
            "body_font_size": round(body_size, 2),
            "headings": headings,
            "tables": tables,
            "page_count": len(doc),
        }
    finally:
        doc.close()


def main():
    """CLI interface: reads PDF from stdin or file argument."""
    import io

    # Capture stdout during PDF processing (PyMuPDF prints warnings there)
    old_stdout = sys.stdout
    sys.stdout = io.StringIO()

    try:
        if len(sys.argv) > 1 and sys.argv[1] != "-":
            # File path provided
            result = extract_structure(pdf_path=sys.argv[1])
        else:
            # Read from stdin
            pdf_bytes = sys.stdin.buffer.read()
            if not pdf_bytes:
                sys.stdout = old_stdout
                print(json.dumps({"error": "No input provided"}))
                sys.exit(1)
            result = extract_structure(pdf_bytes=pdf_bytes)
    finally:
        # Restore stdout
        sys.stdout = old_stdout

    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
