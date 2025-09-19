# Edge Cases and Special Formatting Tests

## Empty Sections

###

### Another Empty Section

## Very Long Headings That Should Test The Parser's Ability To Handle Extended Text In Headers Without Breaking Or Causing Performance Issues When Processing

This section has an extremely long heading to test edge cases in heading extraction and display.

## Unicode and Special Characters

### 中文标题 (Chinese Heading)

### العنوان العربي (Arabic RTL Heading)

### 🚀 Emoji in Heading 🎉

### Heading with Special Characters: !@#$%^&*()_+-=[]{}|;':",.<>?/~`

### Математические символы: ∑∏∫√∞≈≠±×÷

## Deeply Nested Sections

### Level 3

#### Level 4

##### Level 5

###### Level 6
This is the deepest level of markdown heading.

####### Level 7 (Invalid - Should be treated as text)
This should not be recognized as a heading.

## Mixed Indentation and Formatting

   ### Heading with Leading Spaces

	### Heading with Tab

  * List with irregular spacing
    - Nested with different bullet
      + Even more nested
        * Fourth level nesting
          - Fifth level (unusual)

1.  Numbered list with extra space
  2. Inconsistent indentation
   3.  More spacing issues
    4.    Even more spaces

## Complex Tables

| Simple | Table |
|--------|-------|
| Cell 1 | Cell 2 |

| Left | Center | Right |
|:-----|:------:|------:|
| L    |   C    |     R |

| Column with | Very Long Header That Spans Multiple Words | Short |
|-------------|---------------------------------------------|-------|
| Regular | This cell contains a lot of text that might cause rendering issues in some parsers or displays | OK |
| | Empty first cell | |
| | | Empty cells |

| Nested | **Bold** | *Italic* | `Code` | [Link](http://example.com) |
|--------|----------|----------|--------|----------------------------|
| <ul><li>List</li><li>In</li><li>Table</li></ul> | ![Image](img.png) | ~~Strike~~ | ==Highlight== | $LaTeX$ |

## Code Blocks with Edge Cases

```
No language specified
```

```python
# Very long line that should test the parser's ability to handle extended code lines without breaking or causing issues in display or processing
def extremely_long_function_name_that_tests_the_limits_of_reasonable_naming_conventions_and_parser_handling(parameter_one_with_long_name, parameter_two_with_even_longer_name, parameter_three_that_continues_this_pattern):
    pass
```

````markdown
```nested
Code block inside code block
```
````

```javascript
// Code with Unicode
const 你好 = "世界";
const مرحبا = "عالم";
console.log(`${你好} ${مرحبا}`);
```

```html
<!-- HTML with problematic characters -->
<script>alert('XSS');</script>
<div onclick="malicious()">Test</div>
```

## Lists with Edge Cases

- Item 1
-
- Item 3 (Item 2 was empty)

1.
2. Second item (First was empty)
3. Third item

- [ ]
- [x] Checked item (Previous was empty)
- [X] Capital X checkbox
- [] Missing space in checkbox
- [ ] Multiple    spaces    in    item

* Mixed
- Bullet
+ Types
  * In same
  - List
  + Structure

1. Numbered
  a. Letter sub-item
    i. Roman numeral
      A. Capital letter
        I. Capital roman

## Blockquotes with Nesting

> Single level quote

> > Double nested quote

> > > Triple nested quote
> > > with multiple lines
> > > in the same level

> Mixed quote
> > Nested part
> Back to first level
> > > Jump to third level

> Quote with **bold**, *italic*, and `code`
>
> Empty line in quote
>
> > Nested after empty

## Links and References

[Regular Link](http://example.com)

[Link with Title](http://example.com "Title text")

[Reference Link][1]

[Case-Insensitive Reference][REF]

[Broken Reference][nowhere]

[Empty Link]()

<http://auto-link.com>

<email@example.com>

http://bare-url-without-brackets.com

[Internal Link](#heading-anchor)

[Link with Special Characters](http://example.com/path?param=value&other=123#anchor)

[1]: http://example.com/one
[ref]: http://example.com/ref
[REF]: http://example.com/should-not-duplicate

## Images with Edge Cases

![Normal Image](image.png)

![](no-alt-text.png)

![Image with Title](image.png "Title text")

![Broken Image](non-existent.png)

![Image with Special Characters in Path](./images/file (1).png)

![SVG Image](image.svg)

![Base64 Image](data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==)

## Horizontal Rules Variations

---

***

___

- - -

* * *

_ _ _

## HTML in Markdown

<div class="custom-class">
  <p>HTML paragraph</p>
  <span style="color: red;">Colored text</span>
</div>

<details>
<summary>Expandable Section</summary>

Hidden content that should be handled properly by the parser.

</details>

<!-- HTML Comment that should be ignored -->

<kbd>Ctrl</kbd> + <kbd>C</kbd>

<mark>Highlighted text</mark>

## Escaping Special Characters

\*Not italic\*

\# Not a heading

\[Not a link\](not-a-url)

\`Not code\`

Backslash at end of line\

Double backslash \\

\\ \\ Multiple \\ Backslashes \\

## Frontmatter Edge Cases

---
empty_value:
null_value: null
number: 42
float: 3.14159
boolean: true
date: 2024-01-20
multiline: |
  This is a multiline
  value in the frontmatter
list:
  - item1
  - item2
nested:
  deeply:
    nested:
      value: test
special_chars: "!@#$%^&*()"
unicode: "你好世界 🌍"
---

## Line Endings and Whitespace

Line with trailing spaces

Line with trailing tabs

Line ending with backslash\
Should be on same paragraph

Line with
Two spaces for break

Line with	tab	characters	inside

## Very Long Lines

This is an extremely long line that continues for a significant amount of text without any line breaks to test how the parser handles very long continuous strings of text that might exceed typical buffer sizes or cause issues with line wrapping or memory allocation in certain parsing implementations and should definitely be handled gracefully without causing any crashes or performance degradation even when the line contains hundreds or thousands of characters in a single continuous stream.

## Edge Case Section Names

### Section named "Constructor"

### Section named "prototype"

### Section named "__proto__"

### Section named "hasOwnProperty"

### Section named "$$special$$"

### Section with    multiple    spaces

### 	Section with tab

### Section-with-dashes

### Section_with_underscores

### Section.with.dots

## Mathematical Notation

Inline math: $E = mc^2$

Display math:
$$
\sum_{i=1}^{n} x_i = \int_{0}^{\infty} f(x) dx
$$

Mixed: The formula $\sqrt{a^2 + b^2}$ represents the hypotenuse.

## Task Lists Complex Cases

- [x] Completed task
- [ ] Incomplete task
  - [x] Nested completed
  - [ ] Nested incomplete
    - [ ] Double nested
- [x] Back to root level

## Footnotes

Here's a sentence with a footnote[^1].

Multiple footnotes in one line[^2] are possible[^3].

[^1]: This is the first footnote.
[^2]: Second footnote with **formatting**.
[^3]: Third footnote with [a link](http://example.com).

## Definition Lists

Term 1
:   Definition 1

Term 2
:   Definition 2a
:   Definition 2b

Complex Term
:   Definition with **bold** and *italic*
:   Another definition with `code`

## Abbreviations

The HTML specification is maintained by the W3C.

*[HTML]: Hyper Text Markup Language
*[W3C]: World Wide Web Consortium

## Custom Containers

::: warning
This is a warning box.
:::

::: tip
This is a tip box.
:::

::: danger
This is a danger box.
:::

## Collapsible Sections

<details>
<summary>Click to expand</summary>

### Hidden heading

Hidden content with **formatting**.

```python
# Hidden code
print("Hidden")
```

</details>

## Including Other Files

<!-- This would typically include another file -->
{{% include "other-file.md" %}}

<<[another-file.md]

{{include: ./included.md}}

## Comments and Metadata

[//]: # (This is a comment)
[//]: # "Another comment style"
[comment]: # (Yet another comment format)

<!-- Regular HTML comment -->

[metadata]: # (key: value)

## Raw LaTeX

\begin{equation}
\frac{\partial u}{\partial t} = \alpha \nabla^2 u
\end{equation}

## Strikethrough Variations

~~Simple strikethrough~~

~Not strikethrough~

~~~Still strikethrough~~~

## Unconventional Markdown

This is ++inserted text++ (not standard)

This is ==highlighted text== (not standard)

Sub~script~ and Super^script^ text

## Zalgo Text

T̸͎̊h̵͉́i̷̮͐s̶̱̾ ̵̜̇ï̶̳s̴̱̾ ̶̣̎Z̴̜̽a̴̭͐l̷̰̈ǵ̶̱o̶̜̍ ̷̱̈ẗ̶́ͅë̶̱́x̸̱̌t̴̰̄

## Right-to-Left Text

العربية: هذا نص باللغة العربية يجب أن يُعرض من اليمين إلى اليسار.

עברית: זהו טקסט בעברית שצריך להיות מוצג מימין לשמאל.

## Bidirectional Text

This is English text مع نص عربي في المنتصف and back to English.

## Zero-Width Characters

This​has​zero​width​spaces​between​words.

This‌has‌zero‌width‌non‌joiners.

## Control Characters

Text with control character

## Malformed Structures

[Unclosed link

![Unclosed image

> Unclosed quote

```
Unclosed code block

* Unclosed
  * Nested
    * List

| Unclosed | Table |
| Cell 1 | Cell 2

## The End

This document contains numerous edge cases and special formatting scenarios that should test the robustness of any markdown parser or query engine.

---

*Testing complete.* 🎯