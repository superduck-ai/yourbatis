---
illustration_id: 07
type: framework
style: notion
language: zh
aspect_ratio: "16:9"
backend: imagegen
---
Use case: infographic-diagram
Title: “变量只在所属作用域内可见”. Show lexical scope with nested frames, not a generic compiler pipeline.
ZONES:
Large outer frame labeled “方法作用域”, with a small parameter tag at its top “params”. Inside this frame, two distinct sibling nested frames, arranged side by side with generous spacing.
Left inner frame title “<if> 作用域”. Inside exact XML snippet “<bind name="keyword" value="params.Keyword"/>”. Below this snippet two simple tags “params 可见” and “keyword 可见”.
Right inner frame title “<foreach> 作用域”. Inside small exact declaration labels “item = userID” and “index = i”. Below, one concise label “params、userID、i 可见”.
Below both inner frames but still within the outer method frame, a clearly separate strip labeled “离开内层后”. Inside: a check icon beside “params 可见”; crossed-out tags “keyword” and “userID / i” beside “不可见”. Connect each inner frame downward to its crossed-out variable tag with an arrow labeled “离开作用域”. The placement MUST show the strip outside both inner scopes, inside the method scope.
Use pale blue fill for parameter tag and very light yellow accents for inner variable tags. The sibling frames do not contain one another. Tiny upward lookup arrow inside one child frame points toward outer params tag, labeled “向外查找”. Do not suggest parent scopes can access child variables.
LABELS: Bottom caption “需要在后面使用的 bind，应移到外层”. Keep code identifiers accurate and all Chinese readable. No claim that trim/where/set create scopes; they are not shown. No runtime variable mutation or memory diagrams.
RELATIONSHIPS: Outer-to-inner visibility, inner-to-outer invisibility; visual nesting must be accurate.

COLORS: White #FFFFFF background, near-black #1A1A1A ink and text, sparse pale blue #A8D4F0 and pale yellow #F9E79F accents. Hex values and color names are rendering guidance only, never visible labels.
STYLE: Notion-style minimalist hand-drawn black line art, slight intentional wobble, clean rounded cards, ample whitespace, no gradients, no shadows, no texture, no watermark, no decorative people. Large readable upright Chinese text, precise monospaced code. Clean composition with generous white space. Simple white background. Main elements positioned by content needs. Use only explicitly requested labels. Technical accuracy is more important than decoration.
ASPECT: 16:9 landscape, high resolution, readable at blog width.
