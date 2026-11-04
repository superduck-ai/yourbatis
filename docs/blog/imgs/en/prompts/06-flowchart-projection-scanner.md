---
illustration_id: 06-flowchart-projection-scanner
type: localization
style: notion
language: en
aspect_ratio: '16:9'
backend: imagegen
reference: ../06-flowchart-projection-scanner.png
---
Use case: text-localization
Asset: English counterpart of the supplied Chinese article illustration.
ZONES / CONTENT / LABELS:
Title "How columns reach Go fields". Left "SQL projection": SELECT id, name AS display_name / FROM users, extracted id and display_name. Center "Go fields": type User struct { ID int64 `db:"id"`; DisplayName string `db:"display_name"` } shown as proper multi-line Go struct with no semicolons needed. Right "Fixed scanner": row.Scan( &result.ID, &result.DisplayName, ) with proper lines. Match id→ID→first scan argument, display_name→DisplayName→second. Top caption "Resolved at generation time". Bottom distinct example SELECT id, name FROM users plus WHERE ... ; captions "Fixed column list", "WHERE may vary". Footer "Resolve mappings, without validating the database schema". Do not add schema introspection.
REQUIRED REFERENCE TREATMENT: The supplied source image determines the visual family and semantic structure. Replace every Chinese label with natural concise English according to the specification. Preserve the concept, code identifiers, arrows and hierarchy. Reflow boxes and spacing as needed for English readability; do not merely squeeze English into Chinese-sized boxes. No Chinese characters remain anywhere. Source code and syntax are not translated. All visible labels should be English or unchanged code. Do not add claims absent from the content.
COLORS: white background #FFFFFF, near-black #1A1A1A ink, restrained pale-blue #A8D4F0 and pale-yellow #F9E79F highlights, matching the source. Hex values and color names are guidance only, never printed.
STYLE: Notion minimalist hand-drawn line art, slight intentional wobble, clear rounded cards, large readable English labels, accurate monospace code, generous whitespace, no gradients, no shadows, no textures, no watermark. Clean composition with generous white space. Main elements positioned by content needs. Preserve technical accuracy over decoration. For the cover use only pale-blue accents.
ASPECT: 16:9 landscape, high resolution.
