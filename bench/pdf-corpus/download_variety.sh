#!/bin/bash
# Download public domain books, textbooks, and classic literature as PDFs
# Sources: Project Gutenberg, Internet Archive, OpenStax, government docs

set -e
cd "$(dirname "$0")"

echo "=== Classic Literature (Project Gutenberg) ==="
# Project Gutenberg mirrors PDFs - using their generated PDFs
declare -A gutenberg=(
  ["alice_in_wonderland"]="https://www.gutenberg.org/files/11/11-pdf.pdf"
  ["pride_and_prejudice"]="https://www.gutenberg.org/files/1342/1342-pdf.pdf"
  ["frankenstein"]="https://www.gutenberg.org/files/84/84-pdf.pdf"
  ["great_gatsby"]="https://www.gutenberg.org/files/64317/64317-pdf.pdf"
  ["sherlock_holmes"]="https://www.gutenberg.org/files/1661/1661-pdf.pdf"
  ["moby_dick"]="https://www.gutenberg.org/files/2701/2701-pdf.pdf"
  ["war_and_peace"]="https://www.gutenberg.org/files/2600/2600-pdf.pdf"
  ["tale_of_two_cities"]="https://www.gutenberg.org/files/98/98-pdf.pdf"
  ["dracula"]="https://www.gutenberg.org/files/345/345-pdf.pdf"
  ["art_of_war"]="https://www.gutenberg.org/files/132/132-pdf.pdf"
  ["metamorphosis"]="https://www.gutenberg.org/files/5200/5200-pdf.pdf"
  ["picture_of_dorian_gray"]="https://www.gutenberg.org/files/174/174-pdf.pdf"
  ["count_of_monte_cristo"]="https://www.gutenberg.org/files/1184/1184-pdf.pdf"
  ["les_miserables"]="https://www.gutenberg.org/files/135/135-pdf.pdf"
  ["divine_comedy"]="https://www.gutenberg.org/files/8800/8800-pdf.pdf"
)

for name in "${!gutenberg[@]}"; do
  url="${gutenberg[$name]}"
  echo "  $name"
  curl -sL -o "lit_${name}.pdf" "$url" &
done
wait

echo ""
echo "=== Government & Technical Reports ==="
# NIST, NASA, other government PDFs (always public domain)
declare -A govt=(
  ["nist_cybersecurity"]="https://nvlpubs.nist.gov/nistpubs/CSWP/NIST.CSWP.04162018.pdf"
  ["nist_ai_risk"]="https://nvlpubs.nist.gov/nistpubs/ai/NIST.AI.100-1.pdf"
  ["nist_zero_trust"]="https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-207.pdf"
  ["nist_password"]="https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-63b.pdf"
  ["nist_crypto"]="https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-175B.pdf"
)

for name in "${!govt[@]}"; do
  url="${govt[$name]}"
  echo "  $name"
  curl -sL -o "govt_${name}.pdf" "$url" &
done
wait

echo ""
echo "=== Open Textbooks ==="
# OpenStax and other open-access textbooks
declare -A textbooks=(
  ["calculus_vol1"]="https://assets.openstax.org/oscms-prodcms/media/documents/CalculusVolume1-OP_0iuGKVm.pdf"
  ["calculus_vol2"]="https://assets.openstax.org/oscms-prodcms/media/documents/CalculusVolume2-OP_VQxBqdl.pdf"
  ["university_physics_v1"]="https://assets.openstax.org/oscms-prodcms/media/documents/UniversityPhysicsVol1-WEB_LHxVXrc.pdf"
  ["intro_statistics"]="https://assets.openstax.org/oscms-prodcms/media/documents/IntroductoryStatistics-OP_i6tAFGa.pdf"
  ["principles_economics"]="https://assets.openstax.org/oscms-prodcms/media/documents/PrinciplesofEconomics3e-WEB_D3UpQ0U.pdf"
  ["us_history"]="https://assets.openstax.org/oscms-prodcms/media/documents/USHistory-WEB_qlGgjWF.pdf"
  ["intro_sociology"]="https://assets.openstax.org/oscms-prodcms/media/documents/IntroductiontoSociology3e-WEB_Ae4TJKs.pdf"
  ["chemistry"]="https://assets.openstax.org/oscms-prodcms/media/documents/Chemistry2e-WEB_sCAM0Tn.pdf"
  ["biology"]="https://assets.openstax.org/oscms-prodcms/media/documents/Biology2e-WEB_u6andsA.pdf"
  ["psychology"]="https://assets.openstax.org/oscms-prodcms/media/documents/Psychology2e-WEB_7KGqrEi.pdf"
)

for name in "${!textbooks[@]}"; do
  url="${textbooks[$name]}"
  echo "  $name"
  curl -sL -o "textbook_${name}.pdf" "$url" &
done
wait

echo ""
echo "=== Summary ==="
ls -1 *.pdf 2>/dev/null | wc -l
echo "total PDFs"
du -sh .
