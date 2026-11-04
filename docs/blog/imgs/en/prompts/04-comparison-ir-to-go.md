---
illustration_id: 04-comparison-ir-to-go
type: localization
style: notion
language: en
aspect_ratio: '16:9'
backend: imagegen
reference: ../04-comparison-ir-to-go.png
---
Use case: text-localization
Asset: English counterpart of the supplied Chinese article illustration.
ZONES / CONTENT / LABELS:
Title "Walk the IR, emit Go". Left "IR tree", right "Generated Go". Preserve exact tree, code, indentation and connections from Text to builder.Text, If to if, Parameter to builder.Arg. The SELECT Text and If are siblings; inner Text and Parameter are children of If. Exact output code: builder.Text("SELECT id FROM users") then if p.Keyword != "" { builder.Text(" WHERE name LIKE "); builder.Arg("p.Keyword", p.Keyword) }, with line breaks not semicolons. Bottom caption "Format and write the file" and sequence "Go source" → "go/format" → "*.sqlmap.gen.go". No SQL execution or database icon.
REQUIRED REFERENCE TREATMENT: The supplied source image determines the visual family and semantic structure. Replace every Chinese label with natural concise English according to the specification. Preserve the concept, code identifiers, arrows and hierarchy. Reflow boxes and spacing as needed for English readability; do not merely squeeze English into Chinese-sized boxes. No Chinese characters remain anywhere. Source code and syntax are not translated. All visible labels should be English or unchanged code. Do not add claims absent from the content.
COLORS: white background #FFFFFF, near-black #1A1A1A ink, restrained pale-blue #A8D4F0 and pale-yellow #F9E79F highlights, matching the source. Hex values and color names are guidance only, never printed.
STYLE: Notion minimalist hand-drawn line art, slight intentional wobble, clear rounded cards, large readable English labels, accurate monospace code, generous whitespace, no gradients, no shadows, no textures, no watermark. Clean composition with generous white space. Main elements positioned by content needs. Preserve technical accuracy over decoration. For the cover use only pale-blue accents.
ASPECT: 16:9 landscape, high resolution.
