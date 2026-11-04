---
type: mixed
density: minimal
style: notion
palette: default
image_count: 13
---

## Illustration 1

**Position**: introducing-yourbatis.md / 流水线总览，替换 Mermaid。
**Purpose**: 展示 XML 和 Go 类型信息如何进入生成器，经过检查与转换，生成 Mapper 并交给运行时执行。
**Visual Content**: 横向 Notion 风格流程图，保留两个输入、三个分析分支以及生成期与运行期边界。
**Filename**: 01-flowchart-yourbatis-pipeline.png
**Prompt**: prompts/01-flowchart-yourbatis-pipeline.md

## Illustration 2

**Position**: introducing-yourbatis.md / 标题后、第一段前。
**Purpose**: 从把 SQL 与动态条件写在一起，引出 yourbatis 的代码生成方式。
**Visual Content**: 动态 SQL 与 Go interface 经 go generate 生成普通 Go 代码，Notion 风格。
**Filename**: 02-infographic-yourbatis-intro-v2.png
**Prompt**: prompts/02-infographic-yourbatis-intro-v2.md
**Revision**: 2，旧版图片与提示词保留。

## Illustration 3

**Position**: introducing-yourbatis.md / 去语法糖与 IR，“元素变少是因为前端做了两件事”之后。
**Purpose**: 展示不同 XML 写法如何归约为少量 IR 节点。
**Visual Content**: where/set 归约为不同配置的 Trim；include 展开为 Text；每个节点保留 Span。
**Filename**: 03-comparison-xml-to-ir.png
**Prompt**: prompts/03-comparison-xml-to-ir.md

## Illustration 4

**Position**: introducing-yourbatis.md / 代码生成：遍历 IR，输出 Go，完整搜索示例之前。
**Purpose**: 用一个条件查询说明 IR 节点如何逐项生成 Go 语句。
**Visual Content**: Text、If、Parameter 组成的小型 IR 树，对应 builder.Text、if、builder.Arg；底部展示 go/format 与输出文件。
**Filename**: 04-comparison-ir-to-go.png
**Prompt**: prompts/04-comparison-ir-to-go.md

## Illustration 5

**Position**: introducing-yourbatis.md / 回填：延迟编号的占位符。
**Purpose**: 两组参数 token 在 Build 时统一编号，省略条件不占号。
**Visual Content**: 两组参数 token 在 Build 时统一编号，省略条件不占号。
**Filename**: 05-flowchart-placeholder-backfill.png
**Prompt**: prompts/05-flowchart-placeholder-backfill.md

## Illustration 6

**Position**: introducing-yourbatis.md / 静态分析：从 SQL 文本中提取投影。
**Purpose**: 投影列通过 db 标签映射为固定扫描目标。
**Visual Content**: 投影列通过 db 标签映射为固定扫描目标。
**Filename**: 06-flowchart-projection-scanner.png
**Prompt**: prompts/06-flowchart-projection-scanner.md

## Illustration 7

**Position**: introducing-yourbatis.md / 作用域链。
**Purpose**: 用嵌套区域展示参数、bind 和循环变量的可见范围。
**Visual Content**: 用嵌套区域展示参数、bind 和循环变量的可见范围。
**Filename**: 07-framework-variable-scope.png
**Prompt**: prompts/07-framework-variable-scope.md

## Illustration 8

**Position**: introducing-yourbatis.md / 开发体验：在 Go 和 XML 之间跳转。
**Purpose**: 展示插件在真实 IDE 中的双向导航入口。
**Visual Content**: 合成用户提供的 Go 与 XML 截图，左侧 Go → XML，右侧 XML → Go。
**Filename**: 08-jetbrains-navigation.png
**Prompt**: prompts/08-jetbrains-navigation.md
**References**: references/08-jetbrains-go.png、references/08-jetbrains-xml.png

## Illustration 9

**Position**: introducing-yourbatis.md / 两个阶段：哪些事在生成期确定。
**Purpose**: 区分生成期确定的契约与运行期变化的参数。
**Visual Content**: 区分生成期确定的契约与运行期变化的参数。
**Filename**: 09-comparison-generation-runtime.png
**Prompt**: prompts/09-comparison-generation-runtime.md

## Illustration 10

**Position**: introducing-yourbatis.md / `${}`：由类型决定能生成什么。
**Purpose**: 用两条路径解释参数绑定与直接拼接文本。
**Visual Content**: 用两条路径解释参数绑定与直接拼接文本。
**Filename**: 10-comparison-binding-interpolation.png
**Prompt**: prompts/10-comparison-binding-interpolation.md

## Illustration 11

**Position**: introducing-yourbatis.md / 目标检查：方言就像 `GOOS`/`GOARCH`。
**Purpose**: 展示生成期拒绝与运行期执行前拒绝的顺序。
**Visual Content**: 展示生成期拒绝与运行期执行前拒绝的顺序。
**Filename**: 11-flowchart-dialect-precheck.png
**Prompt**: prompts/11-flowchart-dialect-precheck.md

## Illustration 12

**Position**: introducing-yourbatis.md / 第一层：XML，以及一个自建的 DOM。
**Purpose**: Mapper XML 转为 DOM，源码位置随节点保留。
**Visual Content**: Mapper XML 转为 DOM，源码位置随节点保留。
**Filename**: 12-flowchart-xml-dom.png
**Prompt**: prompts/12-flowchart-xml-dom.md

## Illustration 13

**Position**: introducing-yourbatis.md / 第二层：SQL 文本中的记号。
**Purpose**: SQL 文本拆为 Text、Parameter 与 TextExpression，并拆出参数选项。
**Visual Content**: SQL 文本拆为 Text、Parameter 与 TextExpression，并拆出参数选项。
**Filename**: 13-flowchart-sql-tokenization.png
**Prompt**: prompts/13-flowchart-sql-tokenization.md
