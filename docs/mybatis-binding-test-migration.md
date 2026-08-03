# MyBatis `BindingTest` 迁移矩阵

本矩阵对应本地 `mybatis-3` commit
`2fa2b38690af8f93e32b40e54f8611707b9d71f1` 中的
`org.apache.ibatis.binding.BindingTest`。原类共有 53 个测试方法。

迁移原则：

- **等价迁移**：保留原测试的外部可观察行为。
- **Go-first 替代**：保留用户意图，但使用命名 Go 参数、显式 SQL 分页、
  `yourbatis.Identifier`、生成期校验或 yield 回调替代 Java/MyBatis 机制。
- **不适用当前契约**：依赖项目明确排除的 Java 动态代理、注解、缓存、
  `selectKey`、嵌套对象图或 map 返回能力，不作为 yourbatis 兼容性声明。

| 状态 | 数量 |
|---|---:|
| 等价迁移 | 7 |
| Go-first 替代 | 14 |
| 不适用当前契约 | 32 |
| 合计 | 53 |

21 个可迁移项各自拥有独立 XML。20 个正向 fixture 经独立
`go:generate sqlmapgen` 生成 Mapper，并运行生成 SQL 断言、SQLite 集成测试和
真实 PostgreSQL 集成测试。B36 是预期生成失败的独立 fixture，不进入正常
`go generate`。

统一入口：

- Mapper 与生成指令：`internal/bindingconformance/mapper.go`
- 独立 XML：`internal/bindingconformance/fixtures/b<ID>.xml`
- 生成 SQL与 fixture 完整性：
  `internal/bindingconformance/mapper_generated_test.go`
- SQLite/PostgreSQL 共用行为：
  `internal/bindingconformance/integration_test.go`
- 数据库入口：`sqlite_integration_test.go`、`postgres_integration_test.go`

## 逐项状态

| ID | Java 测试方法 | 状态 | yourbatis 处理 |
|---|---|---|---|
| B01 | `shouldSelectBlogWithPostsUsingSubSelect` | 不适用当前契约 | 依赖 association、collection 与嵌套 select。 |
| B02 | `shouldFindPostsInList` | 等价迁移 | `b02.xml`；slice 固定索引绑定，断言参数名、值和三行结果。 |
| B03 | `shouldFindPostsInArray` | 等价迁移 | `b03.xml`；Go array 固定索引绑定。 |
| B04 | `shouldFindThreeSpecificPosts` | Go-first 替代 | `b04.xml`；多个命名参数与显式 `LIMIT/OFFSET` 替代 `RowBounds`。 |
| B05 | `shouldInsertAuthorWithSelectKey` | 不适用当前契约 | `selectKey` 和参数对象回写被明确禁止。 |
| B06 | `verifyErrorMessageFromSelectKey` | 不适用当前契约 | 不执行隐藏的 `selectKey` 查询。 |
| B07 | `verifyErrorMessageFromInsertAfterSelectKey` | Go-first 替代 | `b07.xml`；直接执行无效 INSERT，断言 `StatementError` 的 statement ID 与 XML 来源。 |
| B08 | `shouldInsertAuthorWithSelectKeyAndDynamicParams` | 不适用当前契约 | 同时依赖 `selectKey`、反射回写与不安全字符串插值。 |
| B09 | `shouldSelectRandom` | Go-first 替代 | `b09.xml`；用可移植非空 scalar 查询验证相同返回契约。 |
| B10 | `shouldExecuteBoundSelectListOfBlogsStatement` | 等价迁移 | `b10.xml`；生成 Mapper 返回 struct slice。 |
| B11 | `shouldExecuteBoundSelectMapOfBlogsById` | 不适用当前契约 | 依赖 `MapKey` 和 map 聚合返回。 |
| B12 | `shouldExecuteMultipleBoundSelectOfBlogsByIdInWithProvidedResultHandlerBetweenSessions` | Go-first 替代 | `b12.xml`；两个生成 Mapper 实例分别使用 yield 回调。 |
| B13 | `shouldExecuteMultipleBoundSelectOfBlogsByIdInWithProvidedResultHandlerInSameSession` | Go-first 替代 | `b13.xml`；同一生成 Mapper 连续执行两个 yield 回调。 |
| B14 | `shouldExecuteMultipleBoundSelectMapOfBlogsByIdInSameSessionWithoutClearingLocalCache` | 不适用当前契约 | 依赖 session local cache 与 `MapKey`。 |
| B15 | `shouldExecuteMultipleBoundSelectMapOfBlogsByIdBetweenTwoSessionsWithGlobalCacheEnabled` | 不适用当前契约 | 依赖 MyBatis 全局 cache。 |
| B16 | `shouldSelectListOfBlogsUsingXMLConfig` | 等价迁移 | `b16.xml`；yourbatis 的正常 XML 生成路径。 |
| B17 | `shouldExecuteBoundSelectListOfBlogsStatementUsingProvider` | 不适用当前契约 | 不支持 Java 注解 SQL provider。 |
| B18 | `shouldExecuteBoundSelectListOfBlogsAsMaps` | 不适用当前契约 | 不支持任意列到 `map[string]any` 的运行时映射。 |
| B19 | `shouldSelectListOfPostsLike` | Go-first 替代 | `b19.xml`；LIKE 参数与显式 SQL 分页。 |
| B20 | `shouldSelectListOfPostsLikeTwoParameters` | Go-first 替代 | `b20.xml`；两个命名 LIKE 参数与显式 SQL 分页。 |
| B21 | `shouldExecuteBoundSelectOneBlogStatement` | 等价迁移 | `b21.xml`；必需单行 struct 查询。 |
| B22 | `shouldExecuteBoundSelectOneBlogStatementWithConstructor` | 不适用当前契约 | 依赖 constructor 和嵌套 select。 |
| B23 | `shouldExecuteBoundSelectBlogUsingConstructorWithResultMap` | 不适用当前契约 | 依赖 constructor 与嵌套 resultMap。 |
| B24 | `shouldExecuteBoundSelectBlogUsingConstructorWithResultMapAndProperties` | 不适用当前契约 | 同上。 |
| B25 | `shouldExecuteBoundSelectBlogUsingConstructorWithResultMapCollection` | 不适用当前契约 | 依赖 constructor、collection 与对象图折叠。 |
| B26 | `shouldExecuteBoundSelectOneBlogStatementWithConstructorUsingXMLConfig` | 不适用当前契约 | constructor 映射明确不支持。 |
| B27 | `shouldSelectOneBlogAsMap` | 不适用当前契约 | 不支持任意 map 行结果。 |
| B28 | `shouldSelectOneAuthor` | 等价迁移 | `b28.xml`；扁平 resultMap 映射所有字段。 |
| B29 | `shouldSelectOneAuthorFromCache` | 不适用当前契约 | 不提供二级缓存和对象实例复用。 |
| B30 | `shouldSelectOneAuthorByConstructor` | 不适用当前契约 | constructor 映射明确不支持。 |
| B31 | `shouldSelectDraftTypedPosts` | 不适用当前契约 | 依赖 discriminator 和运行时子类型。 |
| B32 | `shouldSelectDraftTypedPostsWithResultMap` | 不适用当前契约 | 同上。 |
| B33 | `shouldReturnANotNullToString` | 不适用当前契约 | Java动态代理 `Object.toString` 行为。 |
| B34 | `shouldReturnANotNullHashCode` | 不适用当前契约 | Java动态代理 `Object.hashCode` 行为。 |
| B35 | `shouldCompareTwoMappers` | 不适用当前契约 | Java动态代理相等性行为。 |
| B36 | `shouldFailWhenSelectingOneBlogWithNonExistentParam` | Go-first 替代 | `b36.xml`；运行前在生成期拒绝未知参数，且不写出生成文件。 |
| B37 | `shouldFailWhenSelectingOneBlogWithNullParam` | 等价迁移 | `b37.xml`；nil 指针绑定为 SQL NULL，必需单行返回包装后的 `sql.ErrNoRows`。 |
| B38 | `shouldFailWhenSelectingOneBlogWithNonExistentNestedParam` | Go-first 替代 | `b38.xml`；显式 Go map 索引缺失时产生零值，不发生绑定错误。 |
| B39 | `shouldSelectBlogWithDefault30ParamNames` | Go-first 替代 | `b39.xml`；用显式 Go 参数名替代遗留 `#{0}`/`#{1}`。 |
| B40 | `shouldSelectBlogWithDefault31ParamNames` | Go-first 替代 | `b40.xml`；用显式 Go 参数名替代 `#{param1}`/`#{param2}`。 |
| B41 | `shouldSelectBlogWithAParamNamedValue` | Go-first 替代 | `b41.xml`；动态列名使用 `yourbatis.Identifier`，值继续参数化。 |
| B42 | `shouldCacheMapperMethod` | 不适用当前契约 | Mapper 方法是静态生成代码，不存在运行时 proxy method cache。 |
| B43 | `shouldGetBlogsWithAuthorsAndPosts` | 不适用当前契约 | 依赖 lazy loading、代理和嵌套 select。 |
| B44 | `shouldGetBlogsWithAuthorsAndPostsEagerly` | 不适用当前契约 | 依赖 association、collection 和 fetch type。 |
| B45 | `executeWithResultHandlerAndRowBounds` | Go-first 替代 | `b45.xml`；yield 回调与显式 SQL 分页。 |
| B46 | `executeWithMapKeyAndRowBounds` | 不适用当前契约 | 依赖 `MapKey` map 返回。 |
| B47 | `executeWithCursorAndRowBounds` | Go-first 替代 | `b47.xml`；同步 yield 流式扫描与显式 SQL 分页。 |
| B48 | `registeredMappers` | 不适用当前契约 | 生成 Mapper 无运行时 registry。 |
| B49 | `shouldMapPropertiesUsingRepeatableAnnotation` | 不适用当前契约 | 不解析 Java repeatable mapping annotations。 |
| B50 | `shouldMapConstructorUsingRepeatableAnnotation` | 不适用当前契约 | 注解和 constructor 均不支持。 |
| B51 | `shouldMapUsingSingleRepeatableAnnotation` | 不适用当前契约 | 不解析 Java mapping annotations。 |
| B52 | `shouldMapWhenSpecifyBothArgAndConstructorArgs` | 不适用当前契约 | 注解合并与 constructor 行为。 |
| B53 | `shouldMapWhenSpecifyBothResultAndResults` | 不适用当前契约 | Java repeatable annotation 合并行为。 |

## 验证要求

迁移完成时必须执行：

```bash
go generate ./internal/bindingconformance
go test ./internal/bindingconformance
YOURBATIS_POSTGRES_DSN='...' go test ./internal/bindingconformance
```

PostgreSQL fixture 使用 connection-local temporary tables；SQLite fixture 使用
独立内存数据库。两者均执行真实生成 Mapper，不以手写 runtime 调用替代。
