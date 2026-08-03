# MyBatis Java 测试迁移矩阵

本矩阵对应本地 `mybatis-3` 工作副本 HEAD
`2fa2b38690af8f93e32b40e54f8611707b9d71f1` 中以下三个测试类：

- `org.apache.ibatis.builder.xml.dynamic.DynamicSqlSourceTest`
- `org.apache.ibatis.builder.ParameterExpressionTest`
- `org.apache.ibatis.scripting.xmltags.XMLScriptBuilderTest`

共计 43 个 Java 测试方法。状态含义：

- **等价迁移**：外部可观察的 SQL、参数或错误行为一致。
- **Go-first 替代**：保留测试意图，但使用 Go 表达式、生成期校验或安全绑定替代 Java/MyBatis 内部机制。
- **明确不支持**：能力刻意不进入 Go-first 方言；负向测试确保它在生成期被拒绝。

汇总：

| 状态 | 数量 |
|---|---:|
| 等价迁移 | 25 |
| Go-first 替代 | 10 |
| 明确不支持 | 8 |
| 合计 | 43 |

端到端迁移标准：

- 25 个“等价迁移”和 10 个“Go-first 替代”均有独立
  `internal/mybatisconformance/javafixtures/<ID>.xml`。
- 其中 30 个正向 fixture 由 `java_fixture_mappers.go` 中独立的
  `//go:generate sqlmapgen ...` 指令生成独立 Go 文件，再断言生成的
  Builder SQL、参数名、参数值和敏感标记。
- 4 个预期失败的可迁移 fixture（P02、P11、P14、X02）不能加入正常
  `go generate ./...`，测试会直接调用同一个 `generator.Generate` 并断言生成期错误。
- D22 被拆成 E11、E12、E13、E22 四个独立生成 fixture；D26 有独立负向
  `d26.xml`。完整动态扩展见
  [动态表达式迁移矩阵](./mybatis-dynamic-sql-test-migration.md)。
- `TestJavaMigrationFixtureInventory` 固定检查基础矩阵和动态扩展合计 62 个
  XML fixture、56 个生成文件。

## DynamicSqlSourceTest（27）

对应测试入口：

- `internal/mybatisconformance/javafixtures/`：每项一个 XML。
- `internal/mybatisconformance/java_fixture_mappers.go`：每项一个 Mapper 和生成指令。
- `internal/mybatisconformance/java_fixture_generated_test.go`：生成结果断言。
- `builder_mybatis_migration_test.go`：额外保留的低层运行时回归，不再作为端到端迁移证据。
- `internal/generator/expression_test.go`：有限兼容表达式及拒绝边界。
- `internal/frontend/xmlmapper/parser_test.go`：`${...}` 类型化 IR 解析。

| ID | Java 测试方法 | 状态 | Go 测试/处理 |
|---|---|---|---|
| D01 | `shouldDemonstrateSimpleExpectedTextWithNoLoopsOrConditionals` | 等价迁移 | `javafixtures/d01.xml` → `TestJavaDynamicSqlSourceShouldDemonstrateSimpleExpectedTextWithNoLoopsOrConditionals`。 |
| D02 | `shouldDemonstrateMultipartExpectedTextWithNoLoopsOrConditionals` | 等价迁移 | `javafixtures/d02.xml` → `TestJavaDynamicSqlSourceShouldDemonstrateMultipartExpectedTextWithNoLoopsOrConditionals`，精确断言节点间空格。 |
| D03 | `shouldConditionallyIncludeWhere` | 等价迁移 | `javafixtures/d03.xml` → `TestJavaDynamicSqlSourceShouldConditionallyIncludeWhere`。 |
| D04 | `shouldConditionallyExcludeWhere` | 等价迁移 | `javafixtures/d04.xml` → `TestJavaDynamicSqlSourceShouldConditionallyExcludeWhere`。 |
| D05 | `shouldConditionallyDefault` | 等价迁移 | `javafixtures/d05.xml` → `TestJavaDynamicSqlSourceShouldConditionallyDefault`。 |
| D06 | `shouldConditionallyChooseFirst` | 等价迁移 | `javafixtures/d06.xml` → `TestJavaDynamicSqlSourceShouldConditionallyChooseFirst`。 |
| D07 | `shouldConditionallyChooseSecond` | 等价迁移 | `javafixtures/d07.xml` → `TestJavaDynamicSqlSourceShouldConditionallyChooseSecond`。 |
| D08 | `shouldTrimWHEREInsteadOfANDForFirstCondition` | 等价迁移 | `javafixtures/d08.xml` → `TestJavaDynamicSqlSourceShouldTrimWHEREInsteadOfANDForFirstCondition`，精确断言空白。 |
| D09 | `shouldTrimWHEREANDWithLFForFirstCondition` | 等价迁移 | `javafixtures/d09.xml` → 同名语义测试，精确断言 LF。 |
| D10 | `shouldTrimWHEREANDWithCRLFForFirstCondition` | 等价迁移 | `javafixtures/d10.xml` → 同名语义测试；用字符引用绕过 XML 行尾归一化，精确断言 CRLF。 |
| D11 | `shouldTrimWHEREANDWithTABForFirstCondition` | 等价迁移 | `javafixtures/d11.xml` → 同名语义测试，精确断言 TAB。 |
| D12 | `shouldTrimWHEREORWithLFForFirstCondition` | 等价迁移 | `javafixtures/d12.xml` → 同名语义测试，精确断言 LF。 |
| D13 | `shouldTrimWHEREORWithCRLFForFirstCondition` | 等价迁移 | `javafixtures/d13.xml` → 同名语义测试，精确断言 CRLF。 |
| D14 | `shouldTrimWHEREORWithTABForFirstCondition` | 等价迁移 | `javafixtures/d14.xml` → 同名语义测试，精确断言 TAB。 |
| D15 | `shouldTrimWHEREInsteadOfORForSecondCondition` | 等价迁移 | `javafixtures/d15.xml` → `TestJavaDynamicSqlSourceShouldTrimWHEREInsteadOfORForSecondCondition`。 |
| D16 | `shouldTrimWHEREInsteadOfANDForBothConditions` | 等价迁移 | `javafixtures/d16.xml` → `TestJavaDynamicSqlSourceShouldTrimWHEREInsteadOfANDForBothConditions`，精确断言两条件间空白。 |
| D17 | `shouldTrimNoWhereClause` | 等价迁移 | `javafixtures/d17.xml` → `TestJavaDynamicSqlSourceShouldTrimNoWhereClause`。 |
| D18 | `shouldTrimSETInsteadOfCOMMAForBothConditions` | 等价迁移 | `javafixtures/d18.xml` → `TestJavaDynamicSqlSourceShouldTrimSETInsteadOfCOMMAForBothConditions`。 |
| D19 | `shouldTrimCommaAfterSET` | 等价迁移 | `javafixtures/d19.xml` → `TestJavaDynamicSqlSourceShouldTrimCommaAfterSET`。 |
| D20 | `shouldTrimNoSetClause` | 等价迁移 | `javafixtures/d20.xml` → `TestJavaDynamicSqlSourceShouldTrimNoSetClause`。 |
| D21 | `shouldIterateOnceForEachItemInCollection` | Go-first 替代 | `javafixtures/d21.xml` → `TestJavaDynamicSqlSourceShouldIterateOnceForEachItemInCollection`；使用 `#{item}` 安全绑定并断言三组参数，刻意不迁移 `${item}` 标识符插值。 |
| D22 | `shouldHandleOgnlExpression` | Go-first 替代 | 拆为 `e11.xml`（`indexOf`）、`e12.xml`（集合和 `in`）、`e13.xml`（惰性三元）及 `e22.xml`（显式安全文本），每项独立生成并断言。 |
| D23 | `shouldSkipForEachWhenCollectionIsEmpty` | 等价迁移 | `javafixtures/d23.xml` → `TestJavaDynamicSqlSourceShouldSkipForEachWhenCollectionIsEmpty`，断言无 open/close/separator 和参数。 |
| D24 | `shouldPerformStrictMatchOnForEachVariableSubstitution` | Go-first 替代 | `javafixtures/d24.xml` → `TestJavaDynamicSqlSourceShouldPerformStrictMatchOnForEachVariableSubstitution`，断言根变量 `uuu["u"]` 与循环变量 `u.ID` 的生成参数名和值。 |
| D25 | `cornerCase_ForeachComesFirst` | 等价迁移 | `javafixtures/d25.xml` → `TestJavaDynamicSqlSourceCornerCaseForeachComesFirst`。 |
| D26 | `shouldMapNullStringsToEmptyStrings` | 明确不支持 | `javafixtures/d26.xml` → `TestJavaDynamicSqlSourceNullStringTextIsUnsupported`；普通或 nil 字符串不能进入 SQL，生成期建议改用 `#{}`。 |
| D27 | `testShrinkWhitespacesInSql` | Go-first 替代 | `javafixtures/d27.xml` → `TestJavaDynamicSqlSourceShrinkWhitespacesInSQLGoFirst`，从生成代码精确确认内部空白按 XML 保留；不提供全局 shrink 配置。 |

## ParameterExpressionTest（14）

对应测试入口：

- `internal/mybatisconformance/javafixtures/` 和生成结果测试：端到端主证据。
- `internal/frontend/xmlmapper/sql_text_test.go`：额外的占位符解析单测。
- `internal/generator/validation_test.go`：额外的 Go 语法及作用域单测。

| ID | Java 测试方法 | 状态 | Go 测试/处理 |
|---|---|---|---|
| P01 | `simpleProperty` | 等价迁移 | `javafixtures/p01.xml` → `TestJavaParameterExpressionSimpleProperty`，断言生成参数路径和值。 |
| P02 | `propertyWithSpacesInside` | Go-first 替代 | `javafixtures/p02.xml` → `TestJavaParameterExpressionPropertyWithSpacesInsideFailsGeneration`；独立调用生成器并断言含空格属性名在生成期拒绝。 |
| P03 | `simplePropertyWithOldStyleJdbcType` | 明确不支持 | 同名 Go 子测试确认 `id:VARCHAR` 被拒绝。 |
| P04 | `oldStyleJdbcTypeWithExtraWhitespaces` | 明确不支持 | 同名 Go 子测试确认带空白的旧 JDBC 写法仍被拒绝。 |
| P05 | `expressionWithOldStyleJdbcType` | 明确不支持 | 同名 Go 子测试确认表达式后的 `:VARCHAR` 被拒绝；不实现 JDBC 类型选项。 |
| P06 | `simplePropertyWithOneAttribute` | Go-first 替代 | `javafixtures/p06.xml` → `TestJavaParameterExpressionSimplePropertyWithOneAttributeGoFirst`，断言 `sensitive=true` 进入生成参数。 |
| P07 | `expressionWithOneAttribute` | Go-first 替代 | `javafixtures/p07.xml` → `TestJavaParameterExpressionExpressionWithOneAttributeGoFirst`，断言 Go 表达式结果和值的敏感标记。 |
| P08 | `simplePropertyWithManyAttributes` | 明确不支持 | 同名 Go 子测试确认未知参数属性在生成期拒绝。 |
| P09 | `expressionWithManyAttributes` | 明确不支持 | 同名 Go 子测试确认表达式后的未知属性被拒绝。 |
| P10 | `simplePropertyWithOldStyleJdbcTypeAndAttributes` | 明确不支持 | 同名 Go 子测试确认 JDBC 与任意属性组合被拒绝。 |
| P11 | `simplePropertyWithSpaceAndManyAttributes` | Go-first 替代 | `javafixtures/p11.xml` → `TestJavaParameterExpressionSimplePropertyWithSpaceAndManyAttributesFailsGeneration`；用受支持选项隔离并断言含空格路径错误。 |
| P12 | `shouldIgnoreLeadingAndTrailingSpaces` | Go-first 替代 | `javafixtures/p12.xml` → `TestJavaParameterExpressionShouldIgnoreLeadingAndTrailingSpacesGoFirst`，断言生成参数名和值。 |
| P13 | `invalidOldJdbcTypeFormat` | 明确不支持 | 同名 Go 子测试确认 `id:` 在生成期报错；不会进入旧 JDBC 语法分支。 |
| P14 | `invalidJdbcTypeOptUsingExpression` | Go-first 替代 | `javafixtures/p14.xml` → `TestJavaParameterExpressionInvalidJDBCTypeOptUsingExpressionFailsGeneration`，解析层保留 `(expression)+` 为原始路径，随后由真实生成流程按非法 Go 参数表达式拒绝。 |

## XMLScriptBuilderTest（2）

对应测试入口：

- `internal/mybatisconformance/mapper_generated_test.go`
- `internal/frontend/xmlmapper/parser_test.go`

| ID | Java 测试方法 | 状态 | Go 测试/处理 |
|---|---|---|---|
| X01 | `shouldWhereInsertWhitespace` | 等价迁移 | `javafixtures/x01.xml` → `TestJavaXMLScriptBuilderShouldWhereInsertWhitespace`，断言生成 SQL 中 `where` 内外相邻动态节点不会粘连。 |
| X02 | `shouldThrowIfUnknownElementFound` | 等价迁移 | `javafixtures/x02.xml` → `TestJavaXMLScriptBuilderShouldThrowIfUnknownElementFoundDuringGeneration`，独立调用生成器并断言 `<otherwize>` 解析错误。 |

## 维护规则

1. 升级参考 `mybatis-3` 版本时，先更新本页记录的 commit，再重新枚举三个 Java 测试类。
2. 新增或删除 Java 测试时必须同步更新行数和汇总；不能用一个宽泛测试隐式宣称多个未列出的行为。
3. “明确不支持”项仍应有负向测试，避免以后被解析器静默接受。
4. “Go-first 替代”必须同时写明替代的用户可观察行为，不能只记录内部实现不同。
5. “等价迁移 / Go-first 替代”必须保留独立 XML fixture；只有预期成功的
   fixture 可以加入正常 `go generate ./...`。
6. 表达式、上下文变量、foreach 作用域和安全文本的扩展证据记录在
   [动态表达式迁移矩阵](./mybatis-dynamic-sql-test-migration.md)。
