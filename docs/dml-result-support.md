# DML 返回值支持契约

yourbatis 将 MyBatis 的 generated-key 和参数回写能力收敛为数据库原生的
DML 返回值，不实现通用主键回写系统。

## 数据库范围

| 数据库 | 支持等级 | DML 返回方式 |
| --- | --- | --- |
| PostgreSQL | 一级支持 | `result="returning"` |
| SQLite | 一级支持 | `result="returning"` 或 `result="lastid"` |
| MySQL | 二级支持 | `result="lastid"` |
| 其他数据库 | 不支持 | 不提供兼容或模拟 |

MariaDB 不自动作为 MySQL 处理。

## Result 模式

| Result 模式 | Mapper 返回类型 | 执行方式 |
| --- | --- | --- |
| `exec` | `error` | `ExecContext` |
| `rows` | `(int64, error)` | `ExecContext` + `RowsAffected` |
| `lastid` | `(int64, error)` | `ExecContext` + `LastInsertId` |
| `returning` | `(T, error)` | `QueryContext` + 单行 Scan |
| `returning` | `([]T, error)` | `QueryContext` + 多行 Scan |

Result 模式必须显式声明。框架不会在 `RETURNING`、`LastInsertId` 和额外
查询之间自动降级。

## RETURNING

```xml
<insert id="Create" result="returning" resultType="CreatedUser">
  INSERT INTO users (name)
  VALUES (#{name})
  RETURNING id, name, created_at
</insert>
```

INSERT、UPDATE、DELETE 都可以使用 `result="returning"`。返回类型可以是
scalar、struct、指针或 slice；struct 支持自动映射和扁平 `resultMap`。

生成器要求返回投影固定且可静态分析：

- 必须存在顶层 `RETURNING`；
- 不支持 `RETURNING *`；
- 不允许 `if`、`choose` 或 `foreach` 改变返回列；
- 所有 alias 必须使用 `AS` 显式声明；没有 `AS` 时只接受简单列引用；
- computed expression 必须使用 `AS` 指定输出列名；
- 重复列、无法映射的字段和类型不匹配会导致生成失败。

单行方法遇到零行时返回 `sql.ErrNoRows`，遇到多行时返回
`yourbatis.ErrTooManyRows`。多行方法在零行时返回空 slice。

## LastInsertId

`result="lastid"` 只允许用于 INSERT 和 `(int64, error)` 方法：

```xml
<insert id="Create" result="lastid">
  INSERT INTO users (name)
  VALUES (#{name})
</insert>
```

SQLite 和 MySQL 支持该模式。PostgreSQL 会在执行 INSERT 之前返回能力
错误，避免出现数据已经写入但 `LastInsertId` 随后失败的部分成功。

## Multi-values CreateMany

跨方言批量插入使用普通生成方法配合 `foreach`，一次执行一条
multi-values `INSERT`：

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
    (#{user.Name}, #{user.Email,sensitive=true}, #{user.Status})
  </foreach>
</insert>
```

该方法适用于 PostgreSQL、SQLite 和 MySQL，返回各数据库驱动报告的总影响
行数。它要求非空输入，不执行自动分块，不建立专用的 prepared-statement
缓存，也不隐式开启事务。调用者负责限制单次参数数量；需要空输入 no-op 时应
在调用 Mapper 前处理。

PostgreSQL/SQLite 若需要返回每一行，可另行声明
`result="returning"` 和 `([]T, error)`，并在 SQL 中写明
`RETURNING`。MySQL 不支持该模式。multi-row `lastid` 不能表示所有生成 ID，
不应作为 `CreateMany` 的返回契约。

## 明确不支持

- `selectKey BEFORE/AFTER`；
- 反射式 `keyProperty` 参数回写；
- `keyColumn` 专用映射；
- Oracle sequence 和 `RETURNING INTO`；
- SQL Server `OUTPUT`；
- 隐式多语句或隐式事务模拟 generated key。

对应的 Go-first 替代方式：

- `useGeneratedKeys` 使用 `result="lastid"`；
- `keyProperty` 改为 Mapper 返回值；
- `keyColumn` 改为 SQL alias 或 `resultMap`；
- 多字段回写改为返回 struct；
- 应用生成 UUID/ID 时，将 ID 作为普通 Mapper 参数传入。

## Dialect 校验

生产运行时使用：

```go
yourbatis.DialectPostgres
yourbatis.DialectSQLite
yourbatis.DialectMySQL
```

`DialectQuestion` 仅用于底层 SQL Builder 和兼容性测试，不代表生产数据库。

生成器可以指定目标数据库：

```text
sqlmapgen -mapper UserMapper -sql ./user_mapper.xml -dialect postgres
```

指定目标后，不兼容的 result 模式在生成期失败；未指定时，在真实 SQL
执行前做运行时能力检查。数据库版本和 driver 的具体限制仍以真实执行
结果为最终依据，不通过带副作用的探测 SQL 判断能力。

## 测试

`internal/returningconformance` 提供独立 XML fixture、生成结果断言以及
真实 SQLite 集成测试。设置 `YOURBATIS_POSTGRES_DSN` 后，同一套生成
Mapper 会运行 PostgreSQL 集成测试，并使用临时表避免修改持久业务表。
设置 `YOURBATIS_MYSQL_DSN` 后，`example/user` 会运行 MySQL 的基础
CRUD、`lastid` 和事务回滚集成测试。
