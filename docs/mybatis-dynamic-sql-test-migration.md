# MyBatis 动态表达式测试迁移矩阵

本矩阵是 [43 项核心矩阵](./mybatis-java-test-migration.md) 的扩展，
固定参考 MyBatis commit
`2fa2b38690af8f93e32b40e54f8611707b9d71f1`。这里记录表达式、
上下文变量、foreach 作用域和安全动态文本的 28 项独立证据。

每个正向条目都有独立 XML、独立 `go:generate` 输入、独立生成文件和
`BoundSQL` 断言。负向条目保留独立 XML，由测试直接调用同一生成器并断言
生成期错误，不进入正常 `go generate`。

| 状态 | 数量 |
|---|---:|
| 等价迁移 | 16 |
| Go-first 替代 | 10 |
| 明确不支持 | 2 |
| 合计 | 28 |

统一入口：

- Mapper 与生成指令：`internal/mybatisconformance/java_fixture_mappers.go`
- XML：`internal/mybatisconformance/javafixtures/<ID>.xml`
- 生成文件：`internal/mybatisconformance/java_<ID>.sqlmap.gen.go`
- 行为断言：`internal/mybatisconformance/java_fixture_generated_test.go`
- SQLite：`internal/mybatisconformance/sqlite_dynamic_integration_test.go`
- 完整性检查：`TestJavaMigrationFixtureInventory`

## 核心表达式与上下文变量

| ID | MyBatis 来源 | 原始能力 | 状态 | Go-first 证据 |
|---|---|---|---|---|
| E01 | `ExpressionEvaluatorTest.shouldCompareStringsReturnTrue` | 字符串比较为真 | 等价迁移 | `e01.xml` → `TestJavaExpressionEvaluatorShouldCompareStringsReturnTrue` |
| E02 | `ExpressionEvaluatorTest.shouldCompareStringsReturnFalse` | 字符串比较为假 | 等价迁移 | `e02.xml` → `TestJavaExpressionEvaluatorShouldCompareStringsReturnFalse` |
| E03 | `ExpressionEvaluatorTest.shouldReturnTrueIfNotNull` | 非 nil truthiness | 等价迁移 | `e03.xml` → `TestJavaExpressionEvaluatorShouldReturnTrueIfNotNull` |
| E04 | `ExpressionEvaluatorTest.shouldReturnFalseIfNull` | nil truthiness | 等价迁移 | `e04.xml` → `TestJavaExpressionEvaluatorShouldReturnFalseIfNull` |
| E05 | `ExpressionEvaluatorTest.shouldReturnTrueIfNotZero` | 非零数字 truthiness | 等价迁移 | `e05.xml` → `TestJavaExpressionEvaluatorShouldReturnTrueIfNotZero` |
| E06 | `ExpressionEvaluatorTest.shouldReturnFalseIfZero` | 整数零为假 | 等价迁移 | `e06.xml` → `TestJavaExpressionEvaluatorShouldReturnFalseIfZero` |
| E07 | `ExpressionEvaluatorTest.shouldReturnFalseIfZeroWithScale` | 浮点零为假 | 等价迁移 | `e07.xml` → `TestJavaExpressionEvaluatorShouldReturnFalseIfZeroWithScale` |
| E08 | `ExpressionEvaluatorTest.shouldIterateOverIterable` | iterable foreach | 等价迁移 | `e08.xml` → `TestJavaExpressionEvaluatorShouldIterateOverIterable` |
| E09 | `submitted/dynsql/DynSqlTest.selectLike`，statement `selectLike` | `_parameter` 与 bind | Go-first 替代 | `e09.xml`；单业务参数在生成期解析别名，使用 Go 字符串拼接 |
| E10 | `submitted/multidb/MultiDbTest.shouldExecuteHsqlQueryWithDynamicIf` | `_databaseId` 条件 | 等价迁移 | `e10.xml`；`DatabaseIDProvider` 在 DB 和事务 executor 间传播 |

## D22 拆分与有限兼容语法

MyBatis `DynamicSqlSourceTest.shouldHandleOgnlExpression` 不再作为一个笼统
的“OGNL 不支持”条目，而拆成四个可独立生成和断言的 fixture。

| ID | MyBatis 来源 | 原始能力 | 状态 | Go-first 证据 |
|---|---|---|---|---|
| E11 | `DynamicSqlSourceTest.shouldHandleOgnlExpression` | `String.indexOf` | Go-first 替代 | `e11.xml`；静态 lowering 为 `yourbatis.StringIndexRune` |
| E12 | 同上 | 集合字面量与 `in` | 等价迁移 | `e12.xml`；生成普通 Go 比较与短路表达式 |
| E13 | 同上 | 三元表达式 | 等价迁移 | `e13.xml`；生成惰性立即函数，只求值选中分支 |
| E22 | 同上 | `${...}` 文本结果 | Go-first 替代 | `e22.xml`；数字直接格式化，字符串分支显式转为 `yourbatis.Fragment` |

## submitted 与 OGNL 边界

| ID | MyBatis 来源 | 原始能力 | 状态 | Go-first 证据 |
|---|---|---|---|---|
| E14 | `submitted/ognlstatic/OgnlStaticTest.shouldGetAUserWithIfNode` | `not in {null, ''}` | Go-first 替代 | `e14.xml`；使用可 nil Go 指针和显式解引用 |
| E15 | `submitted/ognl_enum/EnumWithOgnlTest.enumWithOgnlDirector` | enum `toString()` | Go-first 替代 | `e15.xml`；静态适配到 Go `String()` |
| E16 | `submitted/dynsql/DynSqlTest.ognlStaticMethodCall` | Java 静态 helper | Go-first 替代 | `e16.xml`；显式包内 Go helper，不解析 `@Class@member` |
| O01 | `OgnlCacheTest.concurrentAccess` | 运行时 OGNL cache 与私有字段 | 明确不支持 | `o01.xml`；生成期拒绝非导出字段，不实现运行时 cache |
| O02 | `OgnlCacheTest.issue2609` | `compareTo` | Go-first 替代 | `o02.xml`；显式导出的 Go `Compare` 方法 |

## foreach、作用域与安全文本

| ID | MyBatis 来源 | 原始能力 | 状态 | Go-first 证据 |
|---|---|---|---|---|
| E17 | `submitted/bind_in_foreach/BindInForeachTest.testBindInForeach` | foreach 内 bind 与 index | Go-first 替代 | `e17.xml`；生成局部变量与包内 Go helper |
| E18 | `submitted/foreach_map/ForEachMapTest.shouldGetStringKeyStringValueEntries` | `map[string]string` key/value | 等价迁移 | `e18.xml` |
| E19 | `submitted/foreach_map/ForEachMapTest.shouldGetIntKeyBoolValueEntries` | `map[int]bool` key/value | 等价迁移 | `e19.xml` |
| E20 | `submitted/foreach_map/ForEachMapTest.shouldGetNestedBeanKeyValueEntries` | map 嵌套字段 | 等价迁移 | `e20.xml` |
| E21 | `submitted/nested/NestedForEachTest.nestedSelect` | 嵌套 foreach 与词法作用域 | 等价迁移 | `e21.xml` |
| E23 | `submitted/dynsql/DynSqlTest.selectSimple` | 动态 schema | Go-first 替代 | `e23.xml`；schema 必须是 `yourbatis.Identifier` 并按 dialect 引用 |
| E24 | `submitted/foreach_map/ForEachMapTest.shouldSubstituteIndexWithKey` | map key 动态列名 | Go-first 替代 | `e24.xml`；key 类型为 `yourbatis.Identifier` |
| E25 | `submitted/multipleiterates/MultipleIteratesTest.shouldGetAUser` | 多段 foreach 与 `${index + 1}` | 等价迁移 | `e25.xml`；index 是静态整数，只走数字格式化 |
| D26 | `DynamicSqlSourceTest.shouldMapNullStringsToEmptyStrings` | nil 字符串 `${id}` 变空串 | 明确不支持 | `d26.xml`；生成期拒绝 `*string` 文本输出并建议 `#{}` |

## 安全边界

- `${...}` 只接受整数、浮点、bool、`yourbatis.Identifier` 和
  `yourbatis.Fragment`。
- 普通 string、指针、`any` 和 interface 在生成期拒绝。
- `Identifier` 分段引用；来源不可信时必须通过 `NewIdentifier` 校验。
- `Fragment` 是显式信任边界，不应用于用户输入。
- interface-backed foreach 在生成期拒绝，不使用反射遍历。
- 运行时包和生成代码不导入 `reflect`；生成器只在读取 struct tag 时使用它。
