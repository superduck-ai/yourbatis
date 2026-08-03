# yourbatis

`yourbatis` 是一个面向 Go 的实验性 SQL Mapper：用 Go interface 定义
Mapper API，用严格的 MyBatis 风格 XML 编写 SQL，再生成普通、可静态检查的
Go 代码。

生成后的程序不会在运行时解析 XML，也不会执行 OGNL。SQL 构建、参数绑定、
结果扫描和 Mapper 调用都已经变成普通 Go 代码。

项目仍处于实验阶段。MyBatis XML 解析器已内置在 `internal/parsing`，
仓库可以作为独立 Go 模块构建，不依赖相邻目录中的本地模块。

## 设计目标

- **Go API 优先**：方法参数、返回类型和查询基数由 Mapper interface 表达。
- **生成代码优先**：结果扫描器和动态 SQL 构建器在生成期确定。
- **SQL 语义显式**：不隐藏额外查询，不自动切换不同的 DML 返回机制。
- **尽早失败**：不支持的 XML、表达式、映射或数据库能力在生成期尽量报错。
- **数据库边界明确**：只支持 PostgreSQL、SQLite 和 MySQL。

## 数据库支持

| 数据库 | 支持级别 | 占位符 | DML `RETURNING` | `LastInsertId` |
| --- | --- | --- | --- | --- |
| PostgreSQL | 一级 | `$1`, `$2`, ... | 支持 | 不支持 |
| SQLite | 一级 | `?` | 支持 | 支持 |
| MySQL | 二级 | `?` | 不支持 | 支持 |

MySQL 的二级支持包括普通 CRUD、事务、动态 SQL、结果映射、受影响行数和
自增 ID。MariaDB 不会被自动当作 MySQL；Oracle、SQL Server 及其他数据库
不在支持范围内。

`DialectQuestion` 只用于底层构建器和一致性测试，不代表生产数据库。

## 快速体验

仓库内已经包含一个完整的 SQLite 示例：

```bash
go generate ./example/user
go test ./example/user
```

生成文件可以在
[`example/user/user_mapper.sqlmap.gen.go`](example/user/user_mapper.sqlmap.gen.go)
中查看。

## 定义 Mapper

Mapper interface 的第一个参数必须是 `context.Context`，其余参数必须命名，
最后一个返回值必须是 `error`。每个方法都要在 XML 中有一个同名 statement。

```go
package user

import "context"

//go:generate go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen \
//  -mapper UserMapper \
//  -sql ./user_mapper.xml \
//  -dialect postgres

type User struct {
    ID    int64  `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

type UserMapper interface {
    GetByID(ctx context.Context, id int64) (User, error)
    FindByEmail(ctx context.Context, email string) (User, bool, error)
    Search(ctx context.Context, keyword string) ([]User, error)
    SearchEach(ctx context.Context, keyword string, yield func(User) error) error
}
```

对应的 XML：

```xml
<mapper namespace="UserMapper">
  <resultMap id="UserResult" type="User">
    <id property="ID" column="id"/>
    <result property="Name" column="name"/>
    <result property="Email" column="email"/>
  </resultMap>

  <select id="GetByID" resultMap="UserResult">
    SELECT id, name, email
    FROM users
    WHERE id = #{id}
  </select>

  <select id="FindByEmail" resultMap="UserResult">
    SELECT id, name, email
    FROM users
    WHERE email = #{email,sensitive=true}
  </select>

  <select id="Search" resultType="User">
    SELECT id, name, email
    FROM users
    <where>
      <if test="keyword != ''">
        name LIKE #{keyword}
      </if>
    </where>
    ORDER BY id
  </select>

  <select id="SearchEach" resultType="User">
    SELECT id, name, email
    FROM users
    WHERE name LIKE #{keyword}
    ORDER BY id
  </select>
</mapper>
```

运行生成器：

```bash
go generate ./...
```

默认输出文件名是 Mapper 名称的 snake_case 加 `.sqlmap.gen.go`，例如
`UserMapper` 生成 `user_mapper.sqlmap.gen.go`。生成文件不应手工修改。

## IDE 导航

仓库的 [`jetbrains-plugin`](jetbrains-plugin/README.md) 提供 GoLand/IntelliJ IDEA 插件。
安装后，可以通过独立的 gutter 图标在 Go Mapper interface/method 与对应的
`<mapper>` 或 `select`/`insert`/`update`/`delete` XML statement 之间双向跳转。
插件不会修改 GoLand 原有的 **Go to Implementation** 行为。

## 创建运行时

把已有的 `*sql.DB` 与明确的生产方言交给 `yourbatis.NewDB`：

```go
runtimeDB := yourbatis.NewDB(
    db,
    yourbatis.DialectPostgres,
    yourbatis.WithDatabaseID("postgres"),
    yourbatis.WithLogger(yourbatis.SlogLogger{}),
)

mapper := user.NewUserMapper(runtimeDB)

current, err := mapper.GetByID(ctx, userID)
```

`WithDatabaseID` 为动态 SQL 提供 `_databaseId`。它与方言是两个概念：
方言决定占位符和能力检查，database ID 用于小范围 SQL 分支。

## SELECT 返回语义

查询基数默认从 Go 返回签名推断：

| Mapper 签名 | 基数 | 0 行 | 1 行 | 多行 |
| --- | --- | --- | --- | --- |
| `(T, error)` | 必须一行 | `sql.ErrNoRows` | `T, nil` | `ErrTooManyRows` |
| `(T, bool, error)` | 可选一行 | `zero, false, nil` | `T, true, nil` | `ErrTooManyRows` |
| `([]T, error)` | 任意多行 | 空 slice | slice | slice |
| `(..., yield func(T) error) error` | 流式多行 | `nil` | 逐行回调 | 逐行回调 |

其中 `[]byte` 被视为单个 scalar，而不是多行结果。

### 可选行

`(T, bool, error)` 用于正常表达“记录可能不存在”：

```go
user, found, err := mapper.FindByEmail(ctx, email)
if err != nil {
    return err
}
if !found {
    // 没有匹配行；这不是查询错误
}
```

可以省略 XML 的 `result`，由签名自动推断；也可以显式写
`result="optional"` 作为文档和生成期校验：

```xml
<select id="FindByEmail" result="optional" resultType="User">
  SELECT id, name, email
  FROM users
  WHERE email = #{email}
</select>
```

### 流式消费

最后一个 Mapper 参数是 `func(T) error` 时，生成器会使用逐行扫描路径：

```go
SearchEach(
    ctx context.Context,
    params SearchUserParams,
    yield func(User) error,
) error
```

`yield` 必须是最后一个参数，方法本身只能返回 `error`，XML 必须省略
`result`，但仍可使用普通查询的 `resultType` 或 `resultMap`。生成器从
`func(T) error` 推断行类型并复用静态 scanner；yield 是控制参数，不会进入
XML 表达式或 SQL 参数绑定，也不会影响唯一业务参数的 `_parameter` 别名。

回调在调用方 goroutine 中串行、同步执行，因此不会先把完整结果集加载到
slice：

```go
err := mapper.SearchEach(ctx, params, func(user User) error {
    return process(user)
})
```

回调返回错误会立即停止迭代并原样返回。nil 回调会在执行查询前返回
`yourbatis.ErrNilYield`。查询、扫描和 `rows.Err()` 错误包装为
`*yourbatis.StatementError`；panic 会在关闭数据库 Rows 后继续传播。

首版没有“成功提前停止”信号。流式路径只保证 yourbatis 不分配完整结果
slice；driver 仍可能自行缓冲数据，且连接会一直占用到 Mapper 方法返回。
在事务中使用时，应在事务回调内部完成消费，不要把处理转移到 goroutine。

### Scalar

单字段结果可以显式使用 `result="scalar"`：

```go
CountByStatus(ctx context.Context, status UserStatus) (int64, error)
```

```xml
<select id="CountByStatus" result="scalar">
  SELECT COUNT(*)
  FROM users
  WHERE status = #{status}
</select>
```

Scalar statement 必须暴露且只暴露一个固定列。单列计算表达式按位置扫描，
不要求提供投影别名；struct 和 `resultMap` 映射中的计算表达式仍需使用显式 `AS` 别名。

## 结果映射

Struct 可以通过字段名、`db`/`sqlmap` tag 或扁平 `resultMap` 映射：

```go
type UserLabel struct {
    UserID      int64  `db:"user_id"`
    DisplayName string `db:"display_name"`
}
```

```xml
<select id="GetLabel" resultType="UserLabel">
  SELECT
    id AS user_id,
    name || ' &lt;' || email || '&gt;' AS display_name
  FROM users
  WHERE id = #{id}
</select>
```

生成器会为每种结果生成固定的 `rows.Scan` 调用，因此 SELECT/RETURNING
投影必须可静态分析：

- 必须显式列出字段，不支持 `SELECT *` 或 `RETURNING *`；
- 所有 alias 必须使用 `AS` 显式声明；没有 `AS` 时只接受 `column` 或
  `table.column` 形式的简单列引用；
- computed expression 必须使用 `AS` 指定输出列名；
- 输出列或 alias 不能重复；
- 动态节点不能改变投影列；
- `resultMap` 只支持直接、导出的 struct 字段；
- `resultMap` 与 `resultType` 不能同时使用；
- 通用 `map` 结果不受支持，请使用静态 struct。

## 参数与安全 SQL 文本

### `#{...}`：绑定值

`#{...}` 始终生成 driver 参数，不把值拼接进 SQL：

```xml
WHERE email = #{email,sensitive=true}
```

当前 Go-first 参数选项只有 `sensitive=true|false`。生成的日志事件会保留
敏感标记，供 Logger 决定是否展示参数；内置 `SlogLogger` 不输出参数值。
`#{...}` 和 `${...}` 表达式内不允许 Go 注释；字符串字面量中的 `//` 和
`/* ... */` 仍按普通文本处理。

### `${...}`：受限文本

`${...}` 不是通用字符串插值，只接受：

- 数字和 bool 表达式；
- `yourbatis.Identifier`；
- 显式标记的 `yourbatis.Fragment`。

```go
schema, err := yourbatis.NewIdentifier(configuredSchema)
order := yourbatis.TrustedFragment("DESC") // 只能来自应用自身控制的文本
```

普通 string、pointer、`any` 和 interface 会在生成期被拒绝。业务输入应使用
`#{...}`；不要把用户输入传给 `TrustedFragment`。

## 动态 SQL

当前支持：

| 元素 | 用途 |
| --- | --- |
| `if` | 条件 SQL |
| `choose` / `when` / `otherwise` | 互斥分支 |
| `foreach` | slice、array、map、string 迭代 |
| `where` | 自动添加 `WHERE` 并清理前导 `AND`/`OR` |
| `set` | 自动添加 `SET` 并清理逗号 |
| `trim` | 自定义 prefix/suffix 与覆盖规则 |
| `bind` | 声明生成期可翻译的 Go 表达式 |
| `include` | 复用当前 Mapper 内的 SQL fragment |

动态测试和 `bind` 值最终会编译成 Go 表达式。新 Mapper 推荐直接使用
`nil`、`&&`、`||`、`!` 和 `len(...)`。为迁移已有 MyBatis XML，编译器也
支持一部分常见写法，例如 `null`、`and`、`or`、`not`、`.size`、
`.isEmpty()`、`in`/`not in` 和惰性三元表达式；它不是完整的 OGNL
解释器。

只有一个 SQL 参数时，可以使用 `_parameter` 作为兼容别名。`_databaseId`
来自 `yourbatis.WithDatabaseID`。

nil slice 或 map 在 `foreach` 中按空集合处理。与 MyBatis 一样，原集合只要
包含至少一个元素就会输出 open 和 close；separator 只在实际产生 SQL 的
循环项之间输出。

如果循环体包含 `if` 或 `choose`，原集合非空并不代表过滤后仍会产生 SQL。
这时 MyBatis 语义会保留 open 和 close，例如产生 `AND id IN ()`。对于
`IN` 条件，应在进入 `foreach` 前过滤集合，并让循环体无条件输出参数；空
集合究竟表示省略条件、匹配零行还是参数错误，应由调用方显式决定。

## DML 返回模式

INSERT、UPDATE 和 DELETE 应显式声明结果模式：

| XML result | Mapper 返回类型 | 执行方式 |
| --- | --- | --- |
| `exec` | `error` | `ExecContext` |
| `rows` | `(int64, error)` | `ExecContext` + `RowsAffected` |
| `lastid` | `(int64, error)` | `ExecContext` + `LastInsertId` |
| `returning` | `(T, error)` | `QueryContext` + 单行扫描 |
| `returning` | `([]T, error)` | `QueryContext` + 多行扫描 |

示例：

```xml
<insert id="Create" result="returning" resultType="CreatedUser">
  INSERT INTO users (name, email)
  VALUES (#{params.Name}, #{params.Email})
  RETURNING id, name, email, created_at
</insert>
```

`result="returning"` 仅支持 PostgreSQL 和 SQLite，可用于 INSERT、UPDATE
和 DELETE。单行返回遇到 0 行时返回 `sql.ErrNoRows`，遇到多行时返回
`yourbatis.ErrTooManyRows`。

`result="lastid"` 只用于 SQLite/MySQL 的 INSERT。PostgreSQL 会在执行
INSERT 前拒绝该模式；MySQL 也会在执行 DML 前拒绝
`result="returning"`，避免语句已经产生副作用后才报告能力错误。

框架不会在 `RETURNING`、`LastInsertId` 和额外查询之间自动降级，也不会：

- 执行隐藏的 `selectKey`；
- 用反射把主键写回任意参数字段；
- 为模拟主键返回而隐式开启事务；
- 为不支持的数据库拼装兼容流程。

完整契约见 [DML 返回值支持](docs/dml-result-support.md)。

## 批量插入

批量插入使用显式的 `CreateMany` Mapper 方法和一条 multi-values
`INSERT`。它不是 JDBC/MyBatis 的 `ExecutorType.BATCH`，也不会把输入切成
多个批次：

```go
CreateMany(
    ctx context.Context,
    params []CreateUserParams,
) (int64, error)
```

```xml
<insert id="CreateMany" result="rows">
  INSERT INTO users (name, email, status)
  VALUES
  <foreach collection="params" item="user" separator=",">
    (
      #{user.Name},
      #{user.Email,sensitive=true},
      #{user.Status}
    )
  </foreach>
</insert>
```

生成的方法把整个非空 slice 编译成一条 SQL，并返回
`RowsAffected`。PostgreSQL 使用 `$n` 占位符，SQLite 和 MySQL 使用 `?`
占位符。`sensitive=true` 等参数元数据会保留到每一行。

当前 `CreateMany` 不自动分块，调用者需要控制单次输入大小，避免超过数据库
或驱动的参数数量、SQL 长度和 packet 限制。输入必须至少包含一项；需要接受
空 slice 的上层 API 应在调用 Mapper 前直接返回。方法不会隐式开启事务；
需要把批量插入和其他操作组成一个原子工作单元时，显式使用
`DB.Transaction` 并用事务 executor 构造 Mapper。

## Runtime 与事务

生成的 Mapper 只依赖 `yourbatis.Executor`。普通调用将 Mapper 绑定到
`*yourbatis.DB`：

```go
runtimeDB := yourbatis.NewDB(rawDB, yourbatis.DialectPostgres)
users := user.NewUserMapper(runtimeDB)

found, err := users.GetByID(ctx, userID)
```

`DB` 和 `Tx` 都实现 `Executor`。事务中需要哪个 Mapper，就使用回调传入的
事务 executor 构造哪个 Mapper，不需要生成 Store：

```go
err := runtimeDB.Transaction(ctx, func(tx yourbatis.Executor) error {
    users := user.NewUserMapper(tx)
    roles := role.NewRoleMapper(tx)

    affected, err := users.UpdateStatus(ctx, userID, user.UserStatusActive)
    if err != nil {
        return err
    }
    if affected != 1 {
        return fmt.Errorf("updated %d rows", affected)
    }
    return roles.Grant(ctx, userID, roleID)
})
```

回调返回 `nil` 时提交，返回错误时回滚，panic 时先回滚再继续 panic。
nil 回调返回 `yourbatis.ErrNilTransactionCallback`。需要设置隔离级别或只读
事务时使用 `TransactionOptions`：

```go
err := runtimeDB.TransactionOptions(
    ctx,
    &sql.TxOptions{Isolation: sql.LevelSerializable},
    func(tx yourbatis.Executor) error {
        return user.NewUserMapper(tx).Transfer(ctx, fromID, toID, amount)
    },
)
```

事务生命周期不能放进单个回调时，可以显式管理公开的 `*yourbatis.Tx`：

```go
tx, err := runtimeDB.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

users := user.NewUserMapper(tx)
roles := role.NewRoleMapper(tx)

if _, err := users.Create(ctx, params); err != nil {
    return err
}
if err := roles.Grant(ctx, userID, roleID); err != nil {
    return err
}
return tx.Commit()
```

Mapper 不会通过 `context.Context` 隐式切换 executor。普通操作传 `DB`，
事务操作传 `Tx`，调用处可以直接看出数据库操作属于哪个事务。

## 错误与日志

执行、driver 和扫描错误会包装为 `*yourbatis.StatementError`，其中包含 Mapper
statement ID、XML 来源和原始错误。可以继续使用 `errors.Is`/`errors.As`：

```go
user, err := mapper.GetByID(ctx, id)
switch {
case errors.Is(err, sql.ErrNoRows):
    // 必须一行的查询没有结果
case errors.Is(err, yourbatis.ErrTooManyRows):
    // SQL 违反了单行/可选行契约
case err != nil:
    // driver、扫描或其他 statement 错误
}
```

实现 `yourbatis.Logger` 可以接收 statement、最终 SQL、参数元数据、耗时和
driver 错误。也可以直接使用基于 `log/slog` 的
`yourbatis.SlogLogger`。

调试时可在创建 `yourbatis.DB` 前设置 `YOURBATIS_DEBUG=1`，无需额外配置
Logger 即可通过标准库 `log` 打印多行日志块，其中包含最终 Bound SQL、按
driver 顺序排列的参数、statement 和耗时：

```bash
YOURBATIS_DEBUG=1 go run ./cmd/server
```

日志会保留 SQL 原有换行并去除 XML 的公共缩进。`true`、`yes` 和 `on`
（忽略大小写）也会开启该功能。SQL 仍保留占位符，不会把参数插值进 SQL；
标记为 `sensitive=true` 的参数值显示为 `<redacted>`。显式配置的
`WithLogger` 会与环境调试日志同时生效，事务也会继承相同配置。

## 生成器命令

生成单个 Mapper：

```bash
go run github.com/superduck-ai/yourbatis/cmd/sqlmapgen \
  -mapper UserMapper \
  -sql ./user_mapper.xml \
  -dialect postgres
```

常用参数：

| 参数 | 说明 |
| --- | --- |
| `-mapper` | Go Mapper interface 名称 |
| `-sql` | Mapper XML 路径 |
| `-dialect` | `postgres`、`sqlite` 或 `mysql` |
| `-out` | 输出文件 |
| `-dir` | Go package 目录，默认当前目录 |
| `-runtime-import` | 生成代码使用的 runtime import path |

指定 `-dialect` 时，不兼容的 DML result mode 会在生成期失败；未指定时，
生成代码会在执行有副作用的语句前做运行时能力检查。

## 严格边界

以下能力不在当前范围内：

- 完整 OGNL 或运行时表达式解释；
- `association`、`collection`、constructor、discriminator 等嵌套对象图；
- 动态改变 SELECT/RETURNING 投影；
- 通用 type handler、JDBC 参数选项、stored procedure/callable statement；
- cache、lazy loading、`parameterMap`；
- MyBatis `selectKey` 和反射式 `keyProperty` 回写；
- 跨文件或跨 Mapper 的 SQL fragment 注册表；
- PostgreSQL `DISTINCT ON` 等尚未覆盖的方言特定投影语法。

XML 是严格方言：未知元素、属性、参数选项和无法静态验证的表达式会直接导致
生成失败，而不是静默忽略。

## 开发与测试

```bash
go generate ./...
go test ./...
go vet ./...
go test -race ./...
```

SQLite 集成测试默认运行。真实数据库测试需要：

```bash
YOURBATIS_POSTGRES_DSN='...' go test ./internal/returningconformance ./example/user
YOURBATIS_MYSQL_DSN='...' go test ./example/user
```

PostgreSQL 和 MySQL fixture 使用临时表，避免修改应用的持久业务表。

更多迁移与一致性资料：

- [DML 返回值支持](docs/dml-result-support.md)
- [MyBatis 动态 SQL 迁移计划](docs/mybatis-dynamic-sql-plan.md)
- [Java 动态 SQL 测试迁移状态](docs/mybatis-java-test-migration.md)
- [表达式与 foreach 扩展迁移状态](docs/mybatis-dynamic-sql-test-migration.md)
- [Java BindingTest 迁移状态](docs/mybatis-binding-test-migration.md)
