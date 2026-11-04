---
illustration_id: 01
type: flowchart
style: notion
palette: default
language: zh
aspect_ratio: '16:9'
backend: imagegen
---
Use case: infographic-diagram
Create one high-resolution landscape 16:9 technical process-flow illustration for a Chinese Go programming article. Notion-style minimalist hand-drawn black ink on pure white. Legible Chinese, accurate technical identifiers. Title: “yourbatis 编译流水线”.

ZONES / LAYOUT:
An airy layout with three horizontal bands that flow downward, all readable at blog width. Use consistent rounded hand-drawn cards, small line icons, thin dark arrows, generous whitespace and margins. Keep all text large. No decorative people, no watermark, no table.
Top band, heading “解析 XML”: five cards in a left-to-right chain:
“Mapper XML” → “XML 解析” with subtitle “DOM · XPath · DTD” → “前端” with subtitle “记号 · 表达式” → “去语法糖” with subtitle “宏展开” → “IR 树” with subtitle “源码位置”. Small secondary annotation under XML parsing: “internal/parsing”.
Middle band, heading “检查与转换”: left source card “Go package” → “go/ast + go/types” → central card “语义分析” with subtitle “名字 · 作用域 · 类型”. The top-band IR tree also connects downward to semantic analysis using a clear routed arrow that does not cross cards. From semantic analysis, fan out rightwards to three vertically stacked cards: “降级” with subtitle “MyBatis 表达式 → go/ast”; “静态分析” with subtitle “投影 → scanner”; “目标检查” with subtitle “方言能力”. All three converge on a single card “代码生成 + gofmt” to their right. No arrow between the three parallel cards.
Bottom band, heading “生成与执行”: the code-generation card connects downward to “*.sqlmap.gen.go”, which connects to “运行时” with subtitle “占位符回填 · 执行”. Small visual distinction: dashed boundary separates generation and runtime, and only the runtime card has a very pale yellow fill.

CONNECTIONS: Single direction arrowheads. Preserve exactly the two inputs and the three parallel analysis outputs. Code generation receives only the three analysis outputs. Output code leads to runtime. Do not make an arrow from runtime back into compilation. Route connectors in open space with right-angle bends or gentle curves, avoiding label collisions.
LABELS: Render only the quoted titles, headings and card texts above, plus the specified technical identifiers. Do not print instruction headings such as ZONES or COLORS.
COLORS: White #FFFFFF background, near-black #1A1A1A ink and text, dark-gray secondary text. Very sparse pastel blue #A8D4F0 accent behind semantic analysis and pale yellow #F9E79F accent behind runtime. Color values and color names are rendering guidance only; do NOT display color names, hex codes or palette labels.
STYLE: Notion-style simple line doodles, slight intentional wobble, restrained outline file/tree/gear icons, no gradients, no shadows, no texture. Handwritten-style headings, highly readable upright Chinese lettering and accurate monospace-like Latin identifiers. Text should be large and prominent. Clean composition with generous white space. Simple white background. Main elements positioned by content needs. Prioritize correct topology and readable labels over decoration.
ASPECT: 16:9 landscape, high resolution, moderate diagram complexity.
