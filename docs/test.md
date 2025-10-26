# 角色

你是我的资深 Go/数据库驱动开发代理，负责在 **clickhouse-go（SDK）** 中落地 **Decimal512** 全链路支持，并保证**向后兼容**与**自动下推/推断**逻辑与 ch-go / ClickHouse Server 的实现保持一致。

---

# 背景（已完成的前置工作）

* **ClickHouse Server** 已支持 `Decimal512`。
* **ch-go**（底层协议库）已原生支持 `Decimal512`（二进制编解码、类型常量、自动列推断、downcast 规则等），关键点：

  * **downcast 规则**：`P < 10 → 32；P < 19 → 64；P < 39 → 128；P < 77 → 256；P < 155 → 512；否则保留 Decimal(P,S)`。
  * **类型与常量**：`proto/column.go:254` 增加 `ColumnTypeDecimal512`；`decimalDowncast()` 增加 `prec < 155 -> ColumnTypeDecimal512`；`proto/decimal.go:15` 定义 `type Decimal512 Int512`。
  * **512 位整数与编解码**：`proto/int512.go` 定义 `Int512/UInt512`，提供 `binUInt512/binPutUInt512`（**64 字节，小端，两补码**）。
  * **列实现**（与 32/64/128/256 风格一致）：`proto/col_decimal512_gen.go`、`proto/col_decimal512_safe_gen.go`、`proto/col_decimal512_unsafe_gen.go`。
  * **自动推断**：`proto/col_auto.go` 中 `Decimal(P,S)` 的 `77 ≤ P < 155 → new(ColDecimal512)`；直指 `ColumnTypeDecimal512` → `new(ColDecimal512)`。
  * **构建与测试**：`go build/test` 在默认与 `purego` 两种模式均通过。
* **go.mod replace** 约定（联调 ch-go）：

  * 本地联调：`replace github.com/ClickHouse/ch-go => ../ch-go`
  * 远端 fork（推荐）：`replace github.com/ClickHouse/ch-go => github.com/mengyu-sxyz/ch-go`

---

# 仓库与 PR 约定（重要）

* 仅使用 fork 仓库：https://github.com/mengyu-sxyz/clickhouse-go

* PR 一律向 fork 提交；在 PR/commit/issue/CI 描述中不要提及主仓库（https://github.com/ClickHouse/clickhouse-go）。

* 如需引用/同步上游，请在本地通过 upstream 远程同步到 fork，再发往 fork；PR 文案中使用“上游”表述，不出现主仓库 URL。

* README/CHANGELOG 如需致谢上游，可用“来源于上游项目 ClickHouse/clickhouse-go”而不含 URL（或以纯文本方式呈现）。

---

# 目标（SDK 侧需要你完成）

在 **clickhouse-go（SDK）** 中：

1. 新增 **Decimal512** 类型与列实现，保持与现有 `Decimal32/64/128/256` 的风格一致。
2. 在 **自动推断/下推** 中接入 **ch-go 同步的 downcast 规则**（见上）。
3. 完成 **编解码、数组/元组/Nullable** 支持、`std` 分支（`database/sql` 适配）读写支持、JSON 反射映射支持。
4. 更新测试与示例，构建矩阵（含 `purego`）均需通过。
5. **向后兼容**：不破坏既有 API；默认自动发现与降级；必要时提供开关/回退策略。

---

# 代码位置（SDK 当前相关文件）

> 你将基于以下文件完成开发（部分）：

* 列实现与编解码（部分）

  * `lib/column/decimal.go`
  * `lib/column/bigint.go`
  * `lib/column/column_gen.go`
  * `lib/column/array_gen.go`
  * `lib/column/tuple.go`
  * `lib/column/json_reflect.go`
  * `lib/column/codegen/column.tpl`
  * `lib/column/codegen/main.go`
* 顶层接口处理（`std` 接口分支含 Decimal 分流）

  * `clickhouse_std.go:426`
  * `clickhouse_std.go:433`
* 测试与示例

  * `tests/decimal_test.go`
  * `tests/nullable_array_test.go`
  * `examples/clickhouse_api/decimal.go`

> 若上述路径与实际工程存在偏差，请以工程内实情为准，但需保留同等能力与覆盖面。

---

# 产出物与验收标准

**产出物**：

* 一个 feature 分支（建议名：`feature/decimal512-support-sdk`）。
* 分阶段 PR（每个 PR 体量可审、含清晰 commit）。
* 新增 `Decimal512` 列类型实现（含 `Safe/Unsafe` 路径、`purego` 兼容）。
* 下推/推断与 `database/sql` 标准接口读写支持。
* 完整的单测/示例与构建脚本更新。
* 迁移/兼容性说明与 README 片段。

**验收标准**：

* 自动推断/下推与 ch-go 一致（见上 `P`→位宽规则）。
* 默认与 `-tags=purego` 下 `go test ./...` 通过（Linux/Darwin, Go≥1.22）。
* 大值/边界值/随机回归用例覆盖完整（含负数、不同 `S`、溢出/舍入/截断行为）。
* `std` 分支读写 `shopspring/decimal`、`big.Int + scale`/字符串 往返正确。
* JSON 反射映射能以字符串/数字方式无损往返（取决于配置）。
* 与既有 Decimal32/64/128/256 共存，不破坏既有行为。

---

# 设计与实现要点（请严格遵循）

1. **类型与列实现**

   * 在 `lib/column/decimal.go` 引入 `Decimal512` 支持；尽量复用既有模板与 `big.Int` 表达，或延用工程内约定的定点数表示（系数 + scale）。
   * 若项目已有 `*big.Int` 背书，优先采用；否则新增内部 `int512` 封装（可参考/使用ch-go，ch-go中已经实现int512），**二进制布局需与 ch-go 一致**：**小端、两补码、64 字节**。
   * 更新/扩展 `codegen`：

     * `lib/column/codegen/column.tpl` 加入 512 分支（与 32/64/128/256 同构）。
     * `lib/column/codegen/main.go` 注册 `Decimal512` 目标，生成 `*_gen.go`。
   * 支持 `Nullable`、`Array(Decimal512)`、`Tuple(...Decimal512...)`。

2. **自动推断/下推**

   * 在反射/类型推断路径（`column_gen.go`、`json_reflect.go` 等）接入 **ch-go 同步规则**：

     * `P < 10 → Decimal32`
     * `P < 19 → Decimal64`
     * `P < 39 → Decimal128`
     * `P < 77 → Decimal256`
     * `P < 155 → Decimal512`
     * 其余：保留 `Decimal(P,S)` 明确声明（不强降位）。
   * 保证：当用户明确声明 `Decimal(P,S)`（`77 ≤ P < 155`）且建表/服务器为 512 列型时，SDK 自动选择 `Decimal512` 列实现。

3. **std 接口与参数绑定**（`clickhouse_std.go`）

   * 在 `426/433` 一带对 `driver.Value`/扫描映射新增 `Decimal512` 分支：

     * 写入：接受 `decimal.Decimal`、`*big.Int + scale`、`string`（`"-?[0-9]+(\.[0-9]+)?"`）三类常用输入；统一转内部 512 系数与 `S` 进行编码。
     * 读取：可返回 `decimal.Decimal`（默认）或用户配置下返回 `string`/`*big.Int + scale`。
   * 溢出/精度策略与既有 Decimal 保持一致（舍入规则、报错路径一致）。

4. **JSON 反射/编解码**

   * 在 `json_reflect.go` 为 `Decimal512` 增加字符串/数字解析分支；**默认建议按字符串解析**以避免 JS number 精度丢失。

5. **兼容性 & 回退**

   * 默认自动发现：当读取到 Server 列型为 `Decimal(…S)` 且 `77 ≤ P < 155` → 选用 `Decimal512`。
   * 可通过环境变量或连接参数（例如 `decimal_compat=256`）强制回退到 256（仅用于故障紧急回退）。
   * 版本策略：**MINOR** bump，changelog 说明新增能力与不兼容风险为 0。

6. **测试矩阵**

   * 新增/扩展：`tests/decimal_test.go`、`tests/nullable_array_test.go`：

     * 编码/解码往返：随机（含负数）× 不同 `S`（0/2/9/18/38/76/154）。
     * 边界：`±(10^P-1) / 10^S`，以及跨位宽边界（`P=9/18/38/76/154/155`）。
     * `Array(Decimal512)`、`Nullable(Decimal512)`、`Tuple(…, Decimal512, …)`。
     * `database/sql` 写入 `decimal.Decimal`/`*big.Int+scale`/`string` 全覆盖。
   * 示例：更新 `examples/clickhouse_api/decimal.go` 覆盖 512 写读样例。
   * `purego` 与默认模式均需 `go test ./...` 通过。

---

# 开发流程（请按阶段交付）

**分支**：`feature/decimal512-support-sdk`

**PR-1：骨架与类型**

* 任务：

  1. codegen 模板/驱动接入 512；生成 `*_decimal512*_gen.go`。
  2. 基础列实现（含 `Append`, `Row`, `Decode/Encode`, `Reset`）。
  3. 读取 server 列型 → 选择 `Decimal512`（仅识别，不改 std 路径）。
* 交付：

  * 代码 diff（关键文件）
  * `go test` 初步用例（最小可用）

**PR-2：std 分支 & 绑定**

* 任务：

  1. `clickhouse_std.go` 写入/读取 512 分支。
  2. JSON 反射/自动推断/下推接入规则。
* 交付：

  * 完整单测（含 `database/sql` 场景）

**PR-3：Nullable/Array/Tuple 与回退开关**

* 任务：

  * 三类复合类型完整支持；连接参数/环境变量回退实现与测试。

**PR-4：示例与文档**

* 任务：

  * 更新 `examples/clickhouse_api/decimal.go`、README/CHANGELOG。

> 每个 PR 需附：动机、设计摘录、兼容性声明、测试覆盖截图。

---

# 约束与风格

* **保持与现有列实现一致的命名、接口与错误风格**。
* **无全局行为变更**：仅在 512 路径中触达必要代码。
* **性能**：`unsafe` 路径与 `purego` 路径均需维持与 256 相近的序列化吞吐。

---

# 开始执行

1. 回应一份 **总计划**（以 checklist 形式）与 **文件改动清单**。
2. 直接提交 **PR-1** 对应的 **第一批代码 diff**（贴出最关键变更的 patch/代码片段）；给出 `go test` 运行指令与预期输出要点。
3. 等我确认后，继续推进 PR-2/3/4。

---

# 依赖与联调说明

* 在本仓库 `go.mod` 中加入：

  * 本地：`replace github.com/ClickHouse/ch-go => ../ch-go`
  * 或远端：`replace github.com/ClickHouse/ch-go => github.com/mengyu-sxyz/ch-go`
* CI 建议新增 `purego` 任务：`go test -tags=purego ./...`

---

# 如需做出的权衡（请先给出建议）

* 内部定点数表达：统一以 `big.Int + scale` 还是引入 `int512` 轻量封装？（目标：与既有 Decimal 家族实现一致）
* `std` 分支返回类型默认值：`decimal.Decimal` vs `string`；是否提供连接参数切换？

> 请现在：先输出计划与 PR-1 的首批变更 patch，随后等待我 Review 后继续。
