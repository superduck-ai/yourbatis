---
illustration_id: 06
type: flowchart
style: notion
language: zh
aspect_ratio: "16:9"
backend: imagegen
---
Use case: infographic-diagram
Title: “列如何进入 Go 结构体”. Explain generated result scanning in three left-to-right zones: SQL projection → compile-time mapping → generated scanner.
ZONES:
Left code card heading “SQL 投影”, exact code on separate lines “SELECT id, name AS display_name” and “FROM users”. Below it two extracted column chips “id” and “display_name”.
Center card heading “Go 字段”, exact code with proper backticks:
type User struct {
    ID int64 `db:"id"`
    DisplayName string `db:"display_name"`
}
Draw two unambiguous colored mapping connectors: SQL id → ID, SQL display_name → DisplayName. The aliases are result column names. This is matching column names to explicit db tags, not consulting the database schema.
Right card heading “固定 scanner”, exact code:
row.Scan(
    &result.ID,
    &result.DisplayName,
)
Arrows from ID to the first Scan argument and DisplayName to the second. Preserve SELECT order. A small label above the mapping arrows says “生成期确定”.
Bottom visual: one compact SQL card “SELECT id, name FROM users”, with the SELECT portion outlined as fixed and a conditional clause below “WHERE ...” in a pale movable-looking card. Two concise captions “列清单固定” and “WHERE 可以变化”. This is a second schematic example, separated from the main alias example.
LABELS: Small footer “只确定映射，不检查数据库 schema”. Large text, especially code tags, quotes and ampersands. No reflected fields, no live database rows, no claims of schema verification.
CONNECTIONS: Clearly route column-to-field and field-to-Scan arrows in order. No crossed arrows or connectors over code.

COLORS: White #FFFFFF background, near-black #1A1A1A ink and text, sparse pale blue #A8D4F0 and pale yellow #F9E79F accents. Hex values and color names are rendering guidance only, never visible labels.
STYLE: Notion-style minimalist hand-drawn black line art, slight intentional wobble, clean rounded cards, ample whitespace, no gradients, no shadows, no texture, no watermark, no decorative people. Large readable upright Chinese text, precise monospaced code. Clean composition with generous white space. Simple white background. Main elements positioned by content needs. Use only explicitly requested labels. Technical accuracy is more important than decoration.
ASPECT: 16:9 landscape, high resolution, readable at blog width.
