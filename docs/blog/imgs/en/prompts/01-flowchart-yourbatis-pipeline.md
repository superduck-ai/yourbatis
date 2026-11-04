---
illustration_id: 01-flowchart-yourbatis-pipeline
type: localization
style: notion
language: en
aspect_ratio: '16:9'
backend: imagegen
reference: ../01-flowchart-yourbatis-pipeline.png
---
Use case: text-localization
Asset: English counterpart of the supplied Chinese article illustration.
ZONES / CONTENT / LABELS:
Title "The yourbatis compilation pipeline". Band 1 "Parse XML": Mapper XML → XML parsing (DOM · XPath · DTD, internal/parsing) → Front end (Tokens · Expressions) → Desugaring (Include expansion) → IR tree (Source locations). Band 2 "Check and transform": Go package → go/ast + go/types → Semantic analysis (Names · Scopes · Types); IR tree also feeds Semantic analysis. Semantic analysis branches to three parallel cards: "Expression lowering" (MyBatis expressions → go/ast), "Static analysis" (Projection → scanner), "Target checks" (Dialect capabilities). These three converge into "Code generation + gofmt". Band 3 "Generate and execute": generated "*.sqlmap.gen.go" → "Runtime" (Placeholder numbering · Execution). Retain all arrows and labels, no cycles. Do not show parallel checks as sequential. Keep dense text readable with short English labels.
REQUIRED REFERENCE TREATMENT: The supplied source image determines the visual family and semantic structure. Replace every Chinese label with natural concise English according to the specification. Preserve the concept, code identifiers, arrows and hierarchy. Reflow boxes and spacing as needed for English readability; do not merely squeeze English into Chinese-sized boxes. No Chinese characters remain anywhere. Source code and syntax are not translated. All visible labels should be English or unchanged code. Do not add claims absent from the content.
COLORS: white background #FFFFFF, near-black #1A1A1A ink, restrained pale-blue #A8D4F0 and pale-yellow #F9E79F highlights, matching the source. Hex values and color names are guidance only, never printed.
STYLE: Notion minimalist hand-drawn line art, slight intentional wobble, clear rounded cards, large readable English labels, accurate monospace code, generous whitespace, no gradients, no shadows, no textures, no watermark. Clean composition with generous white space. Main elements positioned by content needs. Preserve technical accuracy over decoration. For the cover use only pale-blue accents.
ASPECT: 16:9 landscape, high resolution.
