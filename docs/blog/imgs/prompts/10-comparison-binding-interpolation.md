---
illustration_id: 10
type: comparison
style: notion
language: zh
aspect_ratio: "16:9"
backend: imagegen
---
Use case: infographic-diagram
Title “参数值与 SQL 文本，走不同的路径”. Two horizontal lanes in a clean Notion diagram, PostgreSQL examples.
ZONES:
Top lane heading “#{}：绑定参数”. A left XML card “WHERE name = #{name}” with small input tag “name: Ada”. Arrow branches to two separate right cards: “SQL” containing exactly “WHERE name = $1”; “参数” containing exactly “[Ada]”. Show Ada stays in parameter list and is not inserted as a quoted literal in SQL. Small caption “交给驱动绑定”. Pale blue accents.
Bottom lane heading “${}：写入 SQL 文本”. Left XML card “ORDER BY ${column}”, input tag “column: yourbatis.Identifier”. Arrow through a small checkpoint “生成期检查类型” to right SQL card with exact content “ORDER BY "name"”. Small caption “按方言引用标识符”. Under checkpoint draw a separate rejected input “普通 string” with a stop cross and label “生成失败”. There must be NO successful path from ordinary string to SQL. Pale yellow accents; red only for rejection.
LABELS: Small badge “PostgreSQL 示例”. Footer “普通值用 #{}；动态列名用 Identifier”. A second small footer “Fragment 会原样写入，只能使用应用控制的文本”. The Identifier example is not arbitrary string interpolation. No generic safety shield suggesting Fragment is automatically sanitized. No numbered placeholder in ${} lane. No question-mark placeholders.
CONNECTIONS: Top input branches into SQL and arguments. Bottom Identifier goes to text output, ordinary string stops before it. These lanes are separate examples, not a pipeline from binding to interpolation.

COLORS: Pure white #FFFFFF background, near-black #1A1A1A ink, sparse pale blue #A8D4F0 and pale yellow #F9E79F highlights. Muted red used only for rejection. Color names and hex values are rendering guidance, never visible labels.
STYLE: Notion minimalist hand-drawn black line art with slight wobble, consistent rounded cards and small simple icons. Large legible upright Chinese text, exact monospaced technical identifiers. Generous whitespace, no gradients, no shadows, no textures, no watermark, no decorative people. Clean composition with generous white space. Simple white background. Main elements positioned by content needs. No tables or grids. Prioritize correct semantics and connectors.
ASPECT: 16:9 landscape, high resolution, legible at blog width.
