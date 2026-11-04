---
illustration_id: 13
type: flowchart
style: notion
language: zh
aspect_ratio: "16:9"
backend: imagegen
---
Use case: infographic-diagram
Title “把 SQL 文本拆成记号”. Show one SQL fragment being split into an ordered sequence of four IR nodes. This is generation-time parsing; not placeholder numbering or execution.
ZONES:
Top full-width source card heading “SQL 文本”. Exact source string on one line if possible:
WHERE email = #{email,sensitive=true} ORDER BY ${column}
Highlight #{email,sensitive=true} in pale blue and ${column} in pale yellow. Plain SQL unfilled. A central downward arrow labeled “扫描并切分”.
Middle ordered horizontal sequence of exactly four rounded cards, with order numbers 1 2 3 4 in small badges:
1. “Text” with exact content “WHERE email = ”
2. “Parameter” with exact content “email”
3. “Text” with exact content “ ORDER BY ”
4. “TextExpression” with exact content “column”
Show spaces at boundaries naturally, no extra printed quote characters required. All four appear in this order, joined with subtle arrows. Do not skip the second Text. Cards 2 and 4 match the source highlights. No ? or $1 placeholders in these nodes.
Bottom-left detail linked only to card 2 “Parameter”: one small parsing breakdown with two fields “路径：email” and “选项：sensitive=true”. Caption “路径和选项分别保存”. The comma separates path and option; do not treat sensitive=true as part of parameter name.
Bottom-right small rule callout with caption “字符串与嵌套括号参与边界识别” and below it “拒绝记号内部注释和未知选项”. No new malformed examples required.
LABELS: Small footer “这里确定记号结构，类型检查在后续阶段完成”. Do not imply ${column} with an arbitrary plain string has passed type checking. Do not show database calls, redaction of SQL source, or runtime XML parsing. Do not automatically quote column as a SQL identifier at this stage.
CONNECTIONS: Source highlights connect clearly to their corresponding node cards, no crossing arrows. Parameter breakdown attaches only to Parameter, not TextExpression. Maintain clean whitespace and legible exact code.

COLORS: Pure white #FFFFFF background, near-black #1A1A1A ink, dark gray secondary text, restrained pastel blue #A8D4F0 and pastel yellow #F9E79F accents. Color values and names are rendering guidance only: do NOT print color names, hex codes, or palette labels.
STYLE: Notion minimalist hand-drawn line art with intentional slight wobble. Rounded document cards, thin black arrows, simple tree and document icons, generous whitespace. Large legible upright Chinese, accurate monospace technical identifiers and code. No gradients, shadows, textures, watermarks or decorative characters. Clean composition with generous white space. Simple white background. Main elements positioned by content needs. No table grid. Prioritize code accuracy and clear relationships.
ASPECT: 16:9 landscape, high resolution, moderate complexity, readable at blog width.
