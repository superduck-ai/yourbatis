---
illustration_id: 11
type: flowchart
style: notion
language: zh
aspect_ratio: "16:9"
backend: imagegen
---
Use case: infographic-diagram
Title “先检查方言能力，再执行写入”. Two independent horizontal lanes showing rejection of result=lastid for PostgreSQL. Use only the database example PostgreSQL, not SQLite or MySQL. This is a capability check before mutations, not a general guarantee that any database error implies no writes.
ZONES:
Top lane heading “指定目标方言”. Left card two lines “result=lastid” and “-dialect postgres”. Arrow to checkpoint “生成期检查”. Arrow to muted-red stop card “生成失败”. Small annotation under stop “不产生可执行的 Mapper”. No database execution card on this lane.
Bottom lane heading “未指定目标方言”. Left card “生成 Mapper”. Arrow to middle card “运行期检查” subtitle “当前方言：postgres”. At this checkpoint show a red stop sign and a branch down to “返回能力错误”. To its right show a pale dashed ghost card “ExecContext” with a crossed-out connecting arrow or barrier BEFORE the card, caption “未执行 INSERT”. The graphic must clearly show ExecContext is NOT called for this unsupported case, no arrow claiming execution happened. Place a faint small database outline to the right of ExecContext, also unreachable in this scenario.
LABELS: Bottom takeaway “拒绝不支持的模式时，写入尚未发生”. Small note “示例：PostgreSQL 不支持 LastInsertId”. No blanket claim all errors are side-effect-free. No hidden fallback query, no selectKey, no automatic transaction, no mutating probes.
CONNECTIONS: Two distinct checks at two different times; top fails generation, bottom checks runtime dialect before execution. Neither lane writes data in this illustrated unsupported case.

COLORS: Pure white #FFFFFF background, near-black #1A1A1A ink, sparse pale blue #A8D4F0 and pale yellow #F9E79F highlights. Muted red used only for rejection. Color names and hex values are rendering guidance, never visible labels.
STYLE: Notion minimalist hand-drawn black line art with slight wobble, consistent rounded cards and small simple icons. Large legible upright Chinese text, exact monospaced technical identifiers. Generous whitespace, no gradients, no shadows, no textures, no watermark, no decorative people. Clean composition with generous white space. Simple white background. Main elements positioned by content needs. No tables or grids. Prioritize correct semantics and connectors.
ASPECT: 16:9 landscape, high resolution, legible at blog width.
