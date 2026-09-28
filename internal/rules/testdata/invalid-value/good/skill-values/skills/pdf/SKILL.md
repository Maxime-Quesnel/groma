---
name: pdf-tools
description: Extracts text from PDF files. Use when the user mentions a PDF.
model: opus[1m]
effort: xhigh
context: fork
agent: Explore
disable-model-invocation: yes
shell: bash
---

Read the PDF with pdftotext.
