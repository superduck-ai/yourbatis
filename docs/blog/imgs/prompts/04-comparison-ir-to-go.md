---
illustration_id: 04
type: comparison
style: notion
palette: default
language: zh
aspect_ratio: '16:9'
backend: imagegen
---
Use case: infographic-diagram
Create one landscape 16:9 high-resolution educational diagram for a Chinese Go programming article. Notion-style minimalist hand-drawn black ink on pure white, matching a friendly technical blog series. Title exactly “遍历 IR，输出 Go”. Show a small accurate IR tree on the left and its corresponding Go statements on the right, using subtle matching highlights and connector arrows. The purpose is explaining code generation rather than executing a query.

ZONES:
Left zone, heading “IR 树”. Draw an unlabeled simple root dot branching to two ordered children: first a card titled “Text” with content “SELECT id FROM users”; second a card titled “If” with condition “p.Keyword != ""”. The If node has two indented child cards, in order: “Text” with content “ WHERE name LIKE ” and “Parameter” with content “p.Keyword”. Use straight or gently bent tree connectors. Indentation must make both lower nodes children of If, and the first SELECT Text a sibling of If.
Right zone, heading “生成的 Go”. A single broad code sheet with the following exact code in legible monospaced lettering, line breaks and indentation preserved:
builder.Text("SELECT id FROM users")
if p.Keyword != "" {
    builder.Text(" WHERE name LIKE ")
    builder.Arg("p.Keyword", p.Keyword)
}
Use small unobtrusive arrows from left SELECT Text to the first builder.Text line, from If to the if line, from child Text to the indented builder.Text line, and from Parameter to builder.Arg. Route connectors in the empty middle gutter; no crossed arrows or arrows over code. Tint Text nodes and corresponding code lines a very pale blue; tint If node and if line pale yellow; keep Parameter unfilled with a black outline. Arrows should be subordinate to the code and tree.
Bottom strip: a simple left-to-right sequence “Go 源码” → “go/format” → “*.sqlmap.gen.go”, centered below the two zones, with a small file icon at the end. A tiny caption above the strip says “格式化后写入文件”. This strip describes generated source formatting, not runtime SQL execution.

LABELS: Render only quoted labels and code supplied above. Exact Go double quotes, parentheses, braces, capitalization and p.Keyword spelling. Text strings are illustrative SQL fragments; no ellipses or extra code lines. No execution result, no database icon, no XML tags, no tables, no runtime interpreter.
COLORS: White #FFFFFF background; near-black #1A1A1A lines and text; dark gray secondary details. Sparse pale blue #A8D4F0 and pale yellow #F9E79F highlights. Color names and hex codes are rendering guidance only: do NOT display color names, hex codes or palette labels.
STYLE: Notion minimalist doodle style, slightly wobbly black outlines, generous whitespace, clean rounded node cards, simple file and tree icons, no people, no gradients, no shadows, no texture, no watermark. Handwritten-style large headings, highly readable upright Chinese text and accurate monospaced Go code. Clean composition with generous white space. Simple white background. Main elements positioned by content needs. Make text large and prominent.
ASPECT: 16:9 landscape; two clear zones plus a slim bottom strip; moderate detail; readable at blog width.
