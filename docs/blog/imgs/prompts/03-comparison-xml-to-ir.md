---
illustration_id: 03
type: comparison
style: notion
palette: default
language: zh
aspect_ratio: '16:9'
backend: imagegen
---
Use case: infographic-diagram
Create a high-resolution landscape 16:9 Chinese technical illustration for a Go SQL mapper article. Notion-style minimalist hand-drawn black line art on pure white. Explain normalization from XML elements to an intermediate representation. Title exactly “从 XML 写法到 IR 节点”. Three airy horizontal lanes, left inputs and right outputs, no table grid. Large clear text and restrained icons.

ZONES:
Top two lanes grouped under a small heading “去语法糖”. First lane: a left card labeled “<where>”, arrow to a right card titled “Trim”, with two readable subtitle lines “前缀 WHERE” and “去掉开头的 AND / OR”. Second lane: a left card labeled “<set>”, arrow to another right card also titled “Trim”, with subtitles “前缀 SET” and “去掉首尾逗号”. A subtle brace or small connecting line beside the two output cards leads to the caption “同一种节点，不同配置”. Keep the two Trim instances separate: they share the node type, not the same instance. Pale blue accent on the two Trim cards only.
Bottom lane under heading “宏展开”: left code card showing exactly two lines:
“<include refid="userColumns"/>”
“<sql id="userColumns">id, name, email</sql>”
Arrow labeled “展开引用” to a right node card titled “Text”, with content “id, name, email”. Pale yellow accent on Text. The referenced sql declaration is a definition, not a literal part of the include text emitted into SQL. The output must contain only the column names.
Bottom margin: a small source-location tag with document-pin icon and text “每个 IR 节点保留 Span”, then a short second line “文件 · 行 · 列”. This is metadata attached to nodes, not another compiler transformation or an executable node. No extra arrows to this footer.

LABELS: Render only the exact quoted labels above. Precise XML angle brackets, tag names and identifiers. No invented code, no nine-node inventory, no generated Go code.
COLORS: Pure white #FFFFFF background. Near-black #1A1A1A text and outlines, dark gray secondary text, sparse pastel blue #A8D4F0 and pale yellow #F9E79F node highlights. Color values and color names are rendering guidance only: do NOT print them, hex codes, or palette labels.
STYLE: Minimalist Notion hand-drawn ink, slight wobble, clean rounded cards, single-weight outlines, small simple document/tree icons. No people, no gradients, no shadows, no textures, no watermark, no decorative sparkles. Handwritten-style headings; upright clearly readable Chinese and monospace-like code. Text large and prominent. Clean composition with generous white space. Simple white background. Main elements positioned by content needs.
ASPECT: 16:9 landscape, moderate complexity, legible at blog width.
