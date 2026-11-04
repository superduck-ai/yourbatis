---
illustration_id: 12
type: flowchart
style: notion
language: zh
aspect_ratio: "16:9"
backend: imagegen
---
Use case: infographic-diagram
Title “先读懂 XML 的结构”. Three left-to-right zones: source XML, XML token stream, simplified DOM tree. This diagram is only the XML parsing layer, not SQL parsing, not IR normalization and not runtime execution.
ZONES:
Left source card heading “Mapper XML”. Exact five code lines with visible line numbers 1 through 5 aligned outside code:
<mapper namespace="UserMapper">
  <select id="Search">
    SELECT id FROM users
  </select>
</mapper>
Middle compact card heading “encoding/xml”, subtitle “token 流”. Three stacked tags “开始元素”, “文本”, “结束元素”, connected by a thin vertical sequence. Label left-to-middle arrow “读取”.
Right heading “DOM 节点树（简化）”. Root node “mapper” with small side attribute tag “namespace: UserMapper”; root child “select” with attribute tag “id: Search”; select child a leaf card “文本” with content “SELECT id FROM users”. Use proper tree connectors. Label middle-to-right arrow “构建 DOM”. The tree is hierarchy, not execution order. Attributes are attached metadata, not sibling elements.
Attach a small pin-shaped tag to select node, label “源码位置：第 2 行”. Under the tree a tiny source metadata note “文件 · 行 · 列”. Arrow from select's source-location tag bends to a small footer note “后续错误可指回 XML”. The diagram does not report an actual error in the valid XML.
Bottom strip, two independent supporting facts with small icons: “XPath 定位 /mapper” and “DTD 内嵌，无需联网”. Do not connect them as additional parsing stages. XML whitespace nodes intentionally omitted from the simplified tree. No parser feature catalog or invented DOM methods. No scanner or sql.Result objects.
LABELS: Only explicitly provided quoted labels and source code. Exact capitalization UserMapper and Search. Label style in Chinese.
CONNECTIONS: XML → token stream → DOM. Root mapper contains select containing SQL text. Location tag tied to select, not to SQL execution.

COLORS: Pure white #FFFFFF background, near-black #1A1A1A ink, dark gray secondary text, restrained pastel blue #A8D4F0 and pastel yellow #F9E79F accents. Color values and names are rendering guidance only: do NOT print color names, hex codes, or palette labels.
STYLE: Notion minimalist hand-drawn line art with intentional slight wobble. Rounded document cards, thin black arrows, simple tree and document icons, generous whitespace. Large legible upright Chinese, accurate monospace technical identifiers and code. No gradients, shadows, textures, watermarks or decorative characters. Clean composition with generous white space. Simple white background. Main elements positioned by content needs. No table grid. Prioritize code accuracy and clear relationships.
ASPECT: 16:9 landscape, high resolution, moderate complexity, readable at blog width.
