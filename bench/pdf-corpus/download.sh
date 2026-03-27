#!/bin/bash
# Download 120 arXiv papers across different CS categories
# These are all text-based PDFs (LaTeX-generated), no OCR needed

set -e

DEST="$(dirname "$0")"
cd "$DEST"

echo "Downloading arXiv papers..."

# CS.AI - Artificial Intelligence (2301.xxxxx)
for i in $(seq 1 20); do
  id=$(printf "2301.%05d" $i)
  echo "  $id"
  curl -sL -o "ai_${id}.pdf" "https://arxiv.org/pdf/${id}.pdf" &
done
wait

# CS.CL - Computation and Language (2302.xxxxx)
for i in $(seq 1 20); do
  id=$(printf "2302.%05d" $i)
  echo "  $id"
  curl -sL -o "cl_${id}.pdf" "https://arxiv.org/pdf/${id}.pdf" &
done
wait

# CS.CV - Computer Vision (2303.xxxxx)
for i in $(seq 1 20); do
  id=$(printf "2303.%05d" $i)
  echo "  $id"
  curl -sL -o "cv_${id}.pdf" "https://arxiv.org/pdf/${id}.pdf" &
done
wait

# CS.LG - Machine Learning (2304.xxxxx)
for i in $(seq 1 20); do
  id=$(printf "2304.%05d" $i)
  echo "  $id"
  curl -sL -o "ml_${id}.pdf" "https://arxiv.org/pdf/${id}.pdf" &
done
wait

# Math/Physics mix (2305.xxxxx)
for i in $(seq 1 20); do
  id=$(printf "2305.%05d" $i)
  echo "  $id"
  curl -sL -o "math_${id}.pdf" "https://arxiv.org/pdf/${id}.pdf" &
done
wait

# Recent papers (2310.xxxxx)
for i in $(seq 1 20); do
  id=$(printf "2310.%05d" $i)
  echo "  $id"
  curl -sL -o "recent_${id}.pdf" "https://arxiv.org/pdf/${id}.pdf" &
done
wait

echo ""
echo "Download complete."
ls -1 *.pdf 2>/dev/null | wc -l
echo "PDFs downloaded."
du -sh .
