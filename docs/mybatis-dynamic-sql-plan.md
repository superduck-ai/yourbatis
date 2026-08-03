# MyBatis 动态 SQL（Go-first）实施计划

## 1. 目标

在不实现完整 OGNL、不引入运行时表达式解释器的前提下，支持
MyBatis 自身测试实际覆盖的动态 SQL 能力，并尽量在 `go generate`
阶段完成解析、作用域检查、类型检查和代码生成。

本计划以本地 `mybatis-3` 工作副本 commit
`2fa2b38690af8f93e32b40e54f8611707b9d71f1` 为固定参考版本。升级
参考版本时，必须先重新盘点上游测试，再调整实现范围。

现有 [MyBatis Java 测试迁移矩阵](./mybatis-java-test-migration.md)
继续作为 43 项基础总账；本计划新增的动态表达式测试必须进入同样可追溯的
迁移矩阵，不能只用生成器内部单元测试代替上游行为证据。

## 实施状态

截至 2026-07-31，本计划的六个阶段已完成第一轮实现：

| 阶段 | 状态 | 主要证据 |
|---|---|---|
| 0. 测试总账 | 已完成 | 43 项核心矩阵、28 项动态扩展矩阵、62 个 XML inventory |
| 1. 编译器骨架 | 已完成 | 原生 Go 优先、有限兼容 parser、带 mode 的统一编译结果 |
| 2. 条件与上下文 | 已完成 | 静态 truthiness、`_parameter`、`_databaseId` 及事务传播 |
| 3. 表达式子集 | 已完成 | 集合、`in/not in`、惰性三元、`indexOf`、Stringer/helper |
| 4. foreach | 已完成 | slice/array/map/string、嵌套作用域、bind、interface 拒绝 |
| 5. 安全文本 | 已完成 | 数字/bool/Identifier/Fragment 白名单，普通字符串生成期拒绝 |
| 6. SQLite 与确定性 | 已完成 | SQLite 执行、事务回滚、databaseId、连续生成摘要一致 |

动态扩展的逐项来源、分类与 fixture 见
[MyBatis 动态表达式测试迁移矩阵](./mybatis-dynamic-sql-test-migration.md)。
后续扩展仍必须重新从阶段 0 的测试迁移门禁开始，不能直接放宽表达式语法。

## 2. 不可破坏的约束

### 2.1 测试先于实现

每项兼容能力必须先完成 MyBatis 测试迁移，再修改生产实现。开发顺序固定为：

1. 锁定 MyBatis commit、Java 测试类和测试方法。
2. 抄录测试意图、输入、期望 SQL、参数及错误行为。
3. 标记为“等价迁移 / Go-first 替代 / 明确不支持”。
4. 建立独立 XML fixture、Mapper 声明和期望断言。
5. 确认测试在功能分支上因缺少目标能力而失败。
6. 才允许修改 parser、IR、generator 或 runtime。
7. 生成代码、行为测试和全仓测试全部通过后才能完成该项。

测试先行描述的是功能分支内的开发顺序，不要求把失败测试单独合并到主分支；
主分支必须始终保持全绿。

以下任一材料缺失时，功能未达到 Definition of Ready，不得开始实现：

- 上游 commit；
- Java 测试文件和测试方法；
- 上游 XML（如果存在）；
- 用户可观察的输入和预期结果；
- 迁移分类及 Go-first 差异说明；
- 独立 yourbatis XML fixture；
- 生成代码或生成错误断言；
- `BoundSQL` 的 SQL、参数名、参数值断言。

### 2.2 每个用例必须独立

每个可迁移 Java 用例必须形成下面的证据链：

```text
MyBatis Java 测试/资源 XML
    → 迁移矩阵条目
    → 独立 yourbatis XML fixture
    → 独立 go:generate 输入
    → 生成的 Mapper/Builder 代码
    → SQL、参数或生成错误断言
```

一个宽泛的 Go 测试不能替代多个未建档的 Java 用例。预期失败的 fixture
不能加入正常的 `go generate ./...`，应由测试直接调用同一个 generator
并断言确定的错误类型和位置。

### 2.3 Go-first 和安全边界

- 原生 Go 表达式是首选语法，MyBatis 拼写只作为有限兼容层。
- 不实现运行时 OGNL evaluator、表达式缓存或反射式属性遍历。
- 动态 SQL 生成代码使用普通 Go `if`、`range`、局部变量和静态字段访问。
- 查询结果继续生成专用 `row.Scan(...)`，不引入反射式结果映射。
- `any` 或 interface 上无法静态证明安全的操作默认在生成期报错。
- 任意 Java 静态类、私有成员、lambda、赋值、`new`、projection/selection
  明确不支持。
- 普通字符串不能通过 `${...}` 直接进入 SQL；动态标识符和片段必须使用
  显式安全类型。

运行时包和生成代码不得导入 `reflect`。生成器目前使用
`reflect.StructTag` 读取 Go 标签是允许的，因为它只发生在
`go generate` 阶段；不得把该许可扩展到运行时表达式求值。

## 3. 上游测试范围

### 3.1 核心测试类

第一批必须完整盘点以下测试，而不是只迁移当前实现容易支持的部分：

| 上游测试 | 主要约束 |
|---|---|
| `builder/xml/dynamic/DynamicSqlSourceTest` | 动态节点、foreach、文本表达式、空集合和空白行为 |
| `builder/xml/dynamic/ExpressionEvaluatorTest` | 字符串比较、null、数字 truthiness、iterable |
| `scripting/xmltags/OgnlCacheTest` | 方法调用和非公开成员的支持边界 |
| `scripting/xmltags/XMLScriptBuilderTest` | XML 动态节点构建和非法节点诊断 |

`DynamicSqlSourceTest`、`ParameterExpressionTest` 和
`XMLScriptBuilderTest` 已进入现有 43 项矩阵。下一步必须先把
`ExpressionEvaluatorTest` 的 8 个测试和 `OgnlCacheTest` 的 2 个测试
加入扩展矩阵，再开始表达式编译器改造。

其中：

- `OgnlCacheTest.concurrentAccess` 依赖运行时 OGNL cache 和私有字段访问，
  预期归类为“明确不支持”，并迁移为生成期拒绝测试。
- `OgnlCacheTest.issue2609` 的 `compareTo` 只迁移测试意图；优先使用
  Go 方法或显式 helper，不开放任意方法调用。
- `DynamicSqlSourceTest.shouldHandleOgnlExpression` 必须拆出
  `indexOf`、集合字面量、`in`、三元表达式和安全文本输出的独立能力证据，
  不能继续只保留一个笼统的“不支持 OGNL”断言。

### 3.2 submitted 集成测试

核心测试迁移完成后，必须盘点所有包含以下内容的
`src/test/resources` XML，并关联到实际执行它的 Java 测试：

```text
<if> <when> <choose> <foreach> <bind> <trim> <where> <set>
${...} _parameter _databaseId @Class@member
```

首批已知测试目录如下：

| 上游目录 | 主要能力 |
|---|---|
| `submitted/dynsql` | bind、`_parameter`、字符串拼接、静态 helper、安全文本 |
| `submitted/bind_in_foreach` | foreach 内 bind、index 算术和局部作用域 |
| `submitted/foreach_map` | map key/value、嵌套值、动态 key |
| `submitted/nested` | 嵌套 foreach、嵌套属性和作用域 |
| `submitted/multipleiterates` | 多段 foreach、index 算术和动态列名 |
| `submitted/ognl_enum` | `_parameter`、enum/Stringer 和方法替代 |
| `submitted/ognlstatic` | 集合字面量、`not in`、Java 静态成员替代 |
| `submitted/multidb` | `_databaseId` 条件 |
| `submitted/member_access` | 非公开成员的拒绝边界 |

该表只是已知入口，不是封闭清单。阶段 0 必须通过仓库扫描形成完整清单；
任何排除项都要在矩阵中说明原因，不能静默跳过。

## 4. 迁移矩阵和 fixture 规范

新增动态表达式条目使用独立 ID，建议从 `E01` 开始。每行至少包含：

| 字段 | 要求 |
|---|---|
| ID | 稳定且不可复用 |
| MyBatis commit | 固定上游版本 |
| Java 测试 | 完整类名和方法名 |
| 上游 XML | 路径和 statement ID；无 XML 时写明程序化 SqlNode |
| 原始表达式 | 原样记录 |
| 测试意图 | 描述外部行为，不描述 OGNL 内部实现 |
| 迁移状态 | 等价迁移 / Go-first 替代 / 明确不支持 |
| Go-first 差异 | 语法、安全性、类型或生命周期差异 |
| yourbatis fixture | 独立 XML 路径 |
| 生成证据 | `go:generate` 指令和生成文件 |
| 行为断言 | SQL、参数、错误和必要的 SQLite 结果 |

建议新增文件布局：

```text
internal/mybatisconformance/
    javafixtures/e01.xml
    javafixtures/e02.xml
    ...
    dynamic_expression_fixture_mappers.go
    dynamic_expression_fixture_generated_test.go
    java_e01.sqlmap.gen.go
    java_e02.sqlmap.gen.go
    ...
```

正向 fixture 至少断言：

- 生成代码包含预期的原生 Go 表达式或控制流；
- `BoundSQL.SQL`；
- 参数数量、顺序、名称和值；
- 敏感标记；
- 多种输入分支，包括 true/false、nil/non-nil、空/非空集合。

负向 fixture 至少断言：

- 在 `go generate` 阶段失败，而不是运行期失败；
- 错误包含 XML 文件、statement ID、属性名和表达式位置；
- 不生成半成品 `.sqlmap.gen.go`；
- 错误文案能给出可执行的 Go-first 替代建议。

## 5. 目标表达式架构

不再继续扩大正则替换链。目标是生成期的小型表达式编译器：

```text
XML expression
    → 优先尝试原生 Go parser
    → MyBatis 兼容子集 lexer/parser
    → 统一 expression AST
    → 作用域和静态类型检查
    → Go AST
    → 普通 Go 源码
```

表达式编译结果至少区分四种上下文：

```go
type ExprMode uint8

const (
    ExprCondition ExprMode = iota // if / when
    ExprValue                     // bind
    ExprCollection                // foreach
    ExprText                      // 受控文本输出
)
```

校验和代码生成必须消费同一份编译结果，不能分别进行文本翻译。

## 6. 实施阶段

每个阶段都必须先完成该阶段涉及的 MyBatis 测试迁移和失败基线，再开始实现。

### 阶段 0：建立动态 SQL 测试总账

测试工作：

1. 固定并记录 `mybatis-3` commit。
2. 枚举核心测试类中的全部测试方法。
3. 扫描所有测试资源 XML 中的动态节点和表达式入口。
4. 将资源 XML 关联到执行它的 Java 测试。
5. 为每项确定迁移状态和理由。
6. 给现有 43 项矩阵增加动态表达式扩展表。
7. 为第一阶段能力建立独立 fixture 和预期断言。

生产实现：无。

退出条件：

- 测试总数、来源和排除项可机器检查；
- `ExpressionEvaluatorTest` 8 项、`OgnlCacheTest` 2 项全部入账；
- D22 不再是不可拆分的笼统条目；
- 每个计划支持的能力都能追溯到至少一个 MyBatis 测试。

### 阶段 1：表达式编译器骨架

先迁移的测试：

- 当前 `null/and/or/not`、单引号、`.size`、`.isEmpty()` 相关 fixture；
- `ExpressionEvaluatorTest` 的字符串比较用例；
- 非法语法、未知变量、未知字段和作用域错误。

实现：

- 原生 Go 表达式优先；
- 增加 tokenizer 和小型 parser；
- 引入统一 expression AST 和 `ExprMode`；
- validation 与 emission 共用编译结果；
- 保持现有生成代码行为不变。

退出条件：

- 现有 fixture 生成结果没有非预期变化；
- 非法表达式在生成期给出带来源位置的错误；
- `go generate ./...`、确定性 diff 和 `go test ./...` 全部通过。

### 阶段 2：条件语义和上下文变量

先迁移的测试：

- `ExpressionEvaluatorTest` 的 null、非 null、零、非零、带 scale 的零；
- `submitted/dynsql` 中 `_parameter` 和 bind；
- `submitted/multidb` 中 `_databaseId`；
- 多参数 Mapper 中 `_parameter` 的 Go-first 拒绝边界。

实现：

- 按静态类型生成 MyBatis truthiness；
- bool 直接使用，数字生成非零判断，可 nil 类型生成 nil 判断；
- 静态类型不确定的 `any`/interface 默认报错；
- 单业务参数时把 `_parameter` 静态解析为该参数；
- 引入独立于 placeholder dialect 的 `DatabaseID` 构建上下文。

退出条件：

- 不引入 `yourbatis.Truth(any)` 反射兜底；
- 运行时和生成代码不导入 `reflect`；
- 所有上下文变量在生成期完成名称和类型解析。

### 阶段 3：MyBatis 测试使用的表达式子集

先迁移的测试：

- `DynamicSqlSourceTest.shouldHandleOgnlExpression` 的拆分 fixture；
- `submitted/ognlstatic` 的集合字面量和 `not in`；
- `submitted/ognl_enum` 的 enum/Stringer 行为；
- `OgnlCacheTest.issue2609` 的 Go-first 比较替代；
- 所有相关负向边界。

实现：

- 集合字面量；
- `in`、`not in`；
- 保持短路和单分支求值的三元表达式 lowering；
- `%` 和测试中已有的字符串拼接；
- 仅对静态类型匹配的 `indexOf`、Stringer/字符串转换提供适配；
- Java `@Class@member` 改为显式 Go package helper 或常量。

退出条件：

- 不开放任意方法调用；
- 三元表达式不会提前执行未选中的分支；
- D22 的每个子能力都有独立 fixture；
- 私有字段和未注册方法仍在生成期被拒绝。

### 阶段 4：foreach 表达式和作用域完善

先迁移的测试：

- `ExpressionEvaluatorTest.shouldIterateOverIterable`；
- `submitted/bind_in_foreach`；
- `submitted/foreach_map`；
- `submitted/nested`；
- `submitted/multipleiterates`；
- `DynamicSqlSourceTest` 的空集合和严格变量替换用例。

实现：

- collection 表达式使用 `ExprCollection`；
- slice、array、map、string 生成原生 `range`；
- map key/value 和嵌套字段静态解析；
- item/index/bind 的词法作用域；
- 空集合不产生 open、close 和 separator；
- 不支持的 interface collection 在生成期报错，不反射遍历。

退出条件：

- 每个可迁移 Java foreach 用例都有独立 XML fixture；
- 生成代码只使用静态类型和原生 `range`；
- 循环变量不会泄漏到循环外层。

### 阶段 5：安全动态文本

先迁移的测试：

- D21 的 `${item}` 和安全绑定替代；
- D22 的文本表达式输出；
- D26 的 null-to-empty 行为边界；
- `submitted/dynsql` 的 schema 和 escaped literal；
- `submitted/foreach_map` 的动态 key；
- `submitted/multipleiterates` 的动态列名。

实现：

- 保留 `#{...}` 作为默认值绑定；
- `${...}` 只允许数字、bool、`yourbatis.Identifier` 和显式
  `yourbatis.Fragment`；
- 普通 string、`any` 和 nil 在生成期拒绝；
- Identifier 由 dialect 正确引用；
- 对可绑定值优先给出改用 `#{...}` 的诊断。

D26 的任意 null-to-empty 字符串插值继续“明确不支持”，除非新的上游测试能
证明它有不可替代的 SQL 构建价值。

退出条件：

- 不存在普通 string 直接拼入 SQL 的路径；
- 所有动态标识符和片段有显式类型；
- 安全替代后的 SQL 意图与上游测试一致。

### 阶段 6：SQLite 集成和生成确定性

先迁移的测试：

- 从前述阶段选择组合覆盖 if/choose/bind/foreach/databaseId 的上游用例；
- 为数据库差异建立 SQLite 的 Go-first 期望。

实现：

- 增加 SQLite 端到端 Mapper；
- 验证生成 SQL 实际执行、事务内执行和回滚；
- 增加生成文件 inventory；
- CI 检查重复生成无 diff。

退出条件：

```bash
go generate ./...
git diff --exit-code
go test ./...
```

全部通过，并且运行时动态 SQL、参数访问和结果映射均无反射。

## 7. 每项功能的完成标准

一项动态 SQL 能力只有同时满足以下条件才算完成：

1. 已进入迁移矩阵，且来源固定到 MyBatis commit 和测试方法。
2. 已明确等价迁移、Go-first 替代或不支持。
3. 每个可迁移 Java 用例都有独立 XML fixture。
4. 正向 fixture 能通过真实 `go:generate` 生成独立代码。
5. 测试断言生成代码和 `BoundSQL`，必要时断言 SQLite 查询结果。
6. 负向 fixture 在生成期产生稳定、可定位、可操作的错误。
7. 生成代码不包含表达式解释器、反射属性访问或反射 foreach。
8. 没有为了支持单个测试而放开更宽泛的任意执行能力。
9. 文档同步说明用户可用语法和明确不支持的边界。
10. 全仓生成、确定性检查和测试通过。

## 8. 明确非目标

本轮不以实现 OGNL 语言兼容为目标，也不以通过 OGNL 项目自身测试为目标。
除非 MyBatis 的动态 SQL 测试证明必要，否则不实现：

- projection、selection；
- lambda；
- 构造对象；
- 表达式赋值和副作用；
- 反射访问私有成员；
- 任意静态 Java 类解析；
- 任意方法调用；
- 运行时表达式解析和缓存；
- 任意字符串 `${...}` 插值。

新增兼容语法必须由 MyBatis 上游测试驱动；仅凭“OGNL 支持这个语法”不能成为
进入实现范围的理由。
