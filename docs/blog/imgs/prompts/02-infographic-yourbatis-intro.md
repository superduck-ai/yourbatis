---
illustration_id: 02
type: infographic
style: notion
palette: default
language: zh
aspect_ratio: '16:9'
backend: imagegen
---
Use case: infographic-diagram
Asset type: opening illustration for a Chinese technical blog introducing yourbatis, a Go dynamic SQL code generator.
Primary request: A friendly, recognizable backend development scene, showing a user-list search with optional filters becoming dynamic SQL, then yourbatis generating ordinary Go code. One coherent visual story in three zones. Notion-style minimalist hand-drawn black line art on white, wide 16:9 landscape, high resolution. Large legible Chinese labels, sparse technical text.

ZONES:
Left third: A hand-drawn browser-window mockup titled “用户列表”. It has a search input labeled “姓名” with value “Ada”, an enabled checkbox labeled “状态：启用”, a simple three-row user list with avatar circles and understated gray text lines, and pagination buttons “1  2  3”. This is a schematic familiar admin page, not a real software screenshot. A small label above the window says “熟悉的筛选需求”. Use a very pale blue highlight for the search and selected status.
Center third: A short arrow from the filters to a single code card titled “动态 SQL”. On this card show the exact short text, line by line:
“SELECT id, name”
“FROM users”
“WHERE name LIKE ?”
“  AND status = ?”
Below the card, two small outlined tags: “姓名可选” and “状态可选”. The WHERE lines may have a very pale yellow highlight. Emphasize that optional filters produce corresponding clauses. No other query text or invented syntax.
Right third: A compact group with title “yourbatis”. Two small document icons labeled “Mapper XML” and “Go interface” converge via two clear arrows into a simple gear labeled “go generate”. The gear connects downward to one output document labeled “Go 代码”, with a small outlined checkmark nearby. Small subtitle below this output: “生成期检查”. Make the two inputs clear; do not draw an arrow implying SQL alone determines the Go interface. A light arrow from the center SQL card to the right Mapper XML document links the narrative.
Title across the top in friendly large hand-lettering: “筛选条件会变，SQL 也会变”. Avoid adding any other slogan or explanatory paragraphs. Keep a shared baseline, balanced zone sizes, generous margins and whitespace. No rigid table borders or vertical dividing rules.

LABELS: Only the exact quoted strings above and pagination numbers. All Chinese characters must be clear and correct. Latin technical identifiers spelled accurately. Do not display instruction names.
COLORS: Pure white #FFFFFF background, near-black #1A1A1A text and linework, dark gray #4A4A4A for secondary details. Tiny soft blue #A8D4F0 and pale yellow #F9E79F highlights, less than 10 percent of the canvas. Color values and color names are rendering guidance only; do NOT display them as visible text.
STYLE: Notion minimalist doodle illustration, slightly wobbly black ink outlines, rounded cards, simple file/browser/gear icons, no people required, no gradients, no shadows, no paper texture, no watermark, no decorative sparkles. Handwritten-style headings, upright highly readable body labels and code. Text should be large and prominent. Clean composition with generous white space. Simple white background. Main elements positioned by content needs.
ASPECT: 16:9, horizontal, three clear zones, moderate detail, highly readable as a blog lead image.
