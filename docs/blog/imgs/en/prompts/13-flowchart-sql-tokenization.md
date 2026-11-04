---
illustration_id: 13-flowchart-sql-tokenization
type: localization
style: notion
language: en
aspect_ratio: '16:9'
backend: imagegen
reference: ../13-flowchart-sql-tokenization.png
---
Use case: text-localization
Asset: English counterpart of the supplied Chinese article illustration.
ZONES / CONTENT / LABELS:
Title "Split SQL text into tokens". Source card "SQL text" with exact string WHERE email = #{email,sensitive=true} ORDER BY ${column}. Arrow label "Scan and split". Four ordered cards exactly Text (WHERE email = ), Parameter (email), Text ( ORDER BY ), TextExpression (column). Below Parameter: "Store path and options separately", "Path: email", "Option: sensitive=true". Rule box "Strings and nested brackets guide boundary detection"; "Reject comments inside tokens and unknown options". Footer "Parse structure here; check types in a later stage". Match source highlights to parameter/expression cards. Do not emit placeholders or quote the column at this parsing stage.
REQUIRED REFERENCE TREATMENT: The supplied source image determines the visual family and semantic structure. Replace every Chinese label with natural concise English according to the specification. Preserve the concept, code identifiers, arrows and hierarchy. Reflow boxes and spacing as needed for English readability; do not merely squeeze English into Chinese-sized boxes. No Chinese characters remain anywhere. Source code and syntax are not translated. All visible labels should be English or unchanged code. Do not add claims absent from the content.
COLORS: white background #FFFFFF, near-black #1A1A1A ink, restrained pale-blue #A8D4F0 and pale-yellow #F9E79F highlights, matching the source. Hex values and color names are guidance only, never printed.
STYLE: Notion minimalist hand-drawn line art, slight intentional wobble, clear rounded cards, large readable English labels, accurate monospace code, generous whitespace, no gradients, no shadows, no textures, no watermark. Clean composition with generous white space. Main elements positioned by content needs. Preserve technical accuracy over decoration. For the cover use only pale-blue accents.
ASPECT: 16:9 landscape, high resolution.
