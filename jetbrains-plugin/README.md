# Yourbatis Mapper Navigation

这是一个面向 GoLand（以及安装了 Go 插件的 IntelliJ IDEA）的 Yourbatis 导航插件。

它在 Go 和 XML 编辑器 gutter 中提供独立的双向导航图标：

- 在 `type UserMapper interface` 的 `UserMapper` 旁点击图标，跳到
  `<mapper namespace="UserMapper">`。
- 在接口方法 `GetByID` 旁点击图标，跳到同一 namespace 下的
  `<select id="GetByID">`、`<insert>`、`<update>` 或 `<delete>`。
- GoLand 原有的 **Go to Implementation** 和绿色 gutter 图标保持不变，仍可跳到
  生成的 `.sqlmap.gen.go` 实现。
- 在 XML 的 `<mapper>` 旁点击 interface 图标，跳到对应的 Go Mapper interface。
- 在 XML 的 `<select>`、`<insert>`、`<update>` 或 `<delete>` 旁点击 method 图标，
  跳到对应的 Go interface method。
- namespace 可以是简单名称，也可以是以接口名结尾的限定名称，例如
  `github.com/acme/user.UserMapper`。

插件使用文件索引定位 XML，不会在每次导航时扫描整个项目；XML 中声明的远程
DTD 也不会在索引阶段被解析或下载。

## 构建

需要 JDK 21。仓库包含 Gradle Wrapper：

```bash
cd jetbrains-plugin
./gradlew test
./gradlew buildPlugin
```

默认针对 GoLand 2025.3（build 253）编译。开发时也可以复用本机 GoLand，避免
下载 IDE：

```bash
./gradlew test -PlocalIdePath=/Applications/GoLand.app
```

插件安装包生成在 `build/distributions/`。在 GoLand 中打开
**Settings | Plugins**，选择齿轮菜单中的 **Install Plugin from Disk...**，然后
选择生成的 ZIP。

## 使用

在 Go Mapper 中点击 XML 图标可以跳到 XML；在 Mapper XML 中点击 interface 或
method 图标可以跳到 Go。若存在多个匹配目标，插件会显示目标选择列表。

安装插件或修改 Mapper XML 后，IDE 会通过正常的项目索引流程自动更新导航结果。
