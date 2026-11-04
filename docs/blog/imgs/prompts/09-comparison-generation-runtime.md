---
illustration_id: 09
type: comparison
style: notion
language: zh
aspect_ratio: "16:9"
backend: imagegen
---
Use case: infographic-diagram
Title “生成期确定规则，运行期处理变化”. Two spacious panels separated by a narrow gutter and one forward arrow. Do not use the full compiler pipeline: this illustration is about timing.
ZONES:
Left panel heading “生成期 · go generate”. At top show two small input document icons “Mapper XML” and “Go interface”. Below use three large cards “字段与表达式类型”, “固定的列与 scanner”, “目标方言能力”. Attach a footnote to the dialect card “指定 -dialect 时”. These all feed into a single output file “*.sqlmap.gen.go” at bottom of left panel.
Right panel heading “运行期 · 每次调用”. Incoming generated file arrow connects to a code-card representing ordinary Go execution. Inside three short labels “参数具体值”, “条件真假与循环次数”, “最终 SQL 与参数顺序”. Beneath, a tiny concrete visual shows same generated function “Search(params)” receiving two alternative input tags “Keyword 为空” and “Keyword = Ada”, leading to two small outputs labeled “不加姓名条件” and “添加姓名条件” respectively. Keep branch arrows unambiguous. This is runtime condition selection, no parsing of XML.
LABELS: Bottom takeaway “生成代码固定，查询内容随参数变化”. Small footer “生成期不检查数据库 schema”. No claim every error is caught in generation, no claim final SQL is static, no loading XML at runtime.
RELATIONSHIPS: Left produces generated Go file once per generation, right executes that code on calls. No runtime feedback arrow to generator. No database verification on left.

COLORS: Pure white #FFFFFF background, near-black #1A1A1A ink, sparse pale blue #A8D4F0 and pale yellow #F9E79F highlights. Muted red used only for rejection. Color names and hex values are rendering guidance, never visible labels.
STYLE: Notion minimalist hand-drawn black line art with slight wobble, consistent rounded cards and small simple icons. Large legible upright Chinese text, exact monospaced technical identifiers. Generous whitespace, no gradients, no shadows, no textures, no watermark, no decorative people. Clean composition with generous white space. Simple white background. Main elements positioned by content needs. No tables or grids. Prioritize correct semantics and connectors.
ASPECT: 16:9 landscape, high resolution, legible at blog width.
