---
illustration_id: 02
type: infographic
style: notion
language: zh
aspect_ratio: '16:9'
backend: imagegen
revision: 2
---
Use case: infographic-diagram
Asset type: replacement opening illustration for a Chinese article introducing yourbatis, a Go dynamic SQL mapper. Create a friendly, clear Notion-style landscape 16:9 illustration. The opening should emphasize writing readable dynamic SQL together and compiling it into ordinary Go, without a cluttered admin dashboard or full compiler pipeline.

ZONES:
Top large friendly hand-lettered title exactly “SQL 写在一起，条件交给 Go”. Small centered subtitle “yourbatis”.
Left half: one large hand-drawn XML document card heading “动态 SQL”. Exact short XML snippet, accurate brackets and quotes:
SELECT id, name FROM users
<where>
  <if test="p.Keyword != ''">
    AND name LIKE #{p.Keyword}
  </if>
</where>
Highlight the if block with a sparse pale-blue patch. Keep the SELECT line and the conditional SQL visibly together in the same document. This is an abbreviated Mapper XML fragment, not generated Go or final SQL.
Below that left card, a small separate input document labeled “Go interface”, and a short line “Search(ctx, p)”. This line is schematic, not a full Go signature. Both the XML card and Go interface card feed via separate arrows into one compact center gear labeled “go generate”.
Right half: a single output document heading “普通 Go 代码” and small filename “*.sqlmap.gen.go”. Inside this document show a simple concise Go sketch exactly:
if p.Keyword != "" {
    builder.Text("AND name LIKE ")
    builder.Arg("p.Keyword", p.Keyword)
}
These lines are a fragment of a builder function, not the complete SQL, so put a tiny label “生成片段” immediately above the code. The XML condition corresponds exactly to the Go condition. Pale-yellow patch behind the Go if line only. Center gear points to this output document. No database execution icon, no fake query result, no manual editing of generated file.
Bottom two small understated labels, spaced apart: “参数与字段提前检查” and “运行时直接执行 Go”. Do not claim all SQL errors are caught or the database schema is checked. No benchmark claims or ORM comparison.

CONNECTIONS: XML and Go interface are the two inputs of go generate; ordinary Go file is output. Exactly these arrows, no runtime back arrows, no missing input, no boxes suggesting interactive runtime XML parsing. Balanced composition with generous whitespace, roughly equal scale code on left and right, broad margins. Minimal simple document/gear icons.
LABELS: Render only specified quoted labels and exact code. Large readable Chinese, exact Keyword, builder, #{} syntax, proper quotes. No palette labels or instructions visible.
COLORS: Pure white #FFFFFF background; near-black #1A1A1A ink and text; dark gray secondary labels. Sparse pastel blue #A8D4F0 and pastel yellow #F9E79F highlights. Color names and hex codes are rendering guidance only, never visible text.
STYLE: Notion minimalist hand-drawn black line illustration with slight wobble, rounded code cards, simple icons, restrained warmth. No photorealism, textures, shadows, gradients, watermarks, decorative people or busy dashboard UI. Handwritten-style headings, upright highly readable Chinese body lettering, accurate monospaced code. Clean composition with generous white space. Simple white background. Main elements positioned by content needs.
ASPECT: 16:9 landscape, high resolution, readable at article width.
