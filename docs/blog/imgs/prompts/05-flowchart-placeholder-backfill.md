---
illustration_id: 05
type: flowchart
style: notion
language: zh
aspect_ratio: "16:9"
backend: imagegen
---
Use case: infographic-diagram
Title: “先收集参数，最后统一编号”. Explain PostgreSQL placeholder numbering with two visually separated horizontal scenarios. Do NOT depict ? placeholders as intermediate tokens: intermediate tokens contain names and values, no placeholder numbers.
ZONES:
Top scenario heading “有筛选条件”. Three stages left to right, headings “收集” → “合并” → “Build() 编号”.
Collect stage has two small parent/child builder cards: parent “Keyword: a%”, child “IDs: 3, 7”, plus small parent card “Limit: 20”. None have numbered placeholders. Merge stage has four consecutive token chips “Keyword”, “ID: 3”, “ID: 7”, “Limit”. Numbering stage has four corresponding chips “$1”, “$2”, “$3”, “$4” in precisely that order. Under them a short SQL card exactly two lines: “WHERE name LIKE $1 AND id IN ($2, $3)” and “LIMIT $4”. Explicitly omit Status and Offset everywhere to keep this a simplified example.
Bottom scenario heading “无筛选条件”. Collect stage shows empty dashed cards “Keyword 为空” and “IDs 为空”, plus “Limit: 20”. Merge stage has only “Limit”. Numbering stage has only “$1”, with a tiny SQL card “LIMIT $1”. No phantom tokens or gaps, no $2 here. Both scenario rows clearly flow left to right independently, not arrows between rows.
LABELS: Bottom takeaway exactly “省略的条件不占号，最终编号保持连续”. Small annotation near Build “PostgreSQL 示例”. No other database or source-code machinery.
CONNECTIONS: Arrows from collect to merge to Build numbering. Match each parameter token to its numbered counterpart; keep labels readable. The point is numbering only after assembly, not a database allocating IDs.

COLORS: White #FFFFFF background, near-black #1A1A1A ink and text, sparse pale blue #A8D4F0 and pale yellow #F9E79F accents. Hex values and color names are rendering guidance only, never visible labels.
STYLE: Notion-style minimalist hand-drawn black line art, slight intentional wobble, clean rounded cards, ample whitespace, no gradients, no shadows, no texture, no watermark, no decorative people. Large readable upright Chinese text, precise monospaced code. Clean composition with generous white space. Simple white background. Main elements positioned by content needs. Use only explicitly requested labels. Technical accuracy is more important than decoration.
ASPECT: 16:9 landscape, high resolution, readable at blog width.
