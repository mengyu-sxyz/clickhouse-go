# Decimal512 Demo 快速开始

## 🚀 5 分钟快速测试

### 1. 修改连接配置

编辑 `main.go` 第 16-21 行：

```go
Addr: []string{"你的ClickHouse地址:9000"},  // 例如: 192.168.1.100:9000
Auth: clickhouse.Auth{
    Database: "你的数据库",     // 例如: test_db
    Username: "你的用户名",     // 例如: admin
    Password: "你的密码",       // 例如: password123
},
```

### 2. 运行测试

```bash
cd /root/clickhouse-go/examples/decimal512_demo
go run main.go
```

或者使用已编译的二进制：

```bash
./decimal512_demo
```

### 3. 预期输出

```
✅ 连接成功！
📊 ClickHouse 版本: 24.8.x

🔨 创建测试表: test_decimal512_demo
✅ 表创建成功

📝 插入测试数据...
✅ 成功插入 3 条数据

🔍 查询数据并验证...
[显示数据表格]

🧪 测试复杂类型支持...
✅ 复杂类型测试通过

🎉 所有测试完成！Decimal512 功能正常工作！
```

## ⚡ 连接配置示例

### 本地 ClickHouse（默认配置）

```go
Addr: []string{"127.0.0.1:9000"},
Auth: clickhouse.Auth{
    Database: "default",
    Username: "default",
    Password: "",
},
```

### Docker ClickHouse

如果你使用 Docker 运行 ClickHouse：

```bash
docker run -d --name clickhouse-server \
  -p 9000:9000 \
  -p 8123:8123 \
  clickhouse/clickhouse-server:24.8
```

然后使用：

```go
Addr: []string{"127.0.0.1:9000"},
Auth: clickhouse.Auth{
    Database: "default",
    Username: "default",
    Password: "",
},
```

### 远程 ClickHouse

```go
Addr: []string{"192.168.1.100:9000"},
Auth: clickhouse.Auth{
    Database: "production",
    Username: "app_user",
    Password: "your_secure_password",
},
```

### ClickHouse 集群（多节点）

```go
Addr: []string{
    "192.168.1.100:9000",
    "192.168.1.101:9000",
    "192.168.1.102:9000",
},
Auth: clickhouse.Auth{
    Database: "cluster_db",
    Username: "cluster_user",
    Password: "cluster_password",
},
```

## 🔧 常见问题

### Q: 连接失败怎么办？

**A:** 检查以下几点：
1. ClickHouse 服务是否运行：`systemctl status clickhouse-server`
2. 端口是否正确（Native 协议是 9000，HTTP 是 8123）
3. 防火墙是否开放端口：`telnet 127.0.0.1 9000`
4. 用户名密码是否正确

### Q: 版本不支持 Decimal512？

**A:** Decimal512 需要 ClickHouse 24.8+，检查版本：

```bash
clickhouse-client --version
```

如果版本低，升级 ClickHouse：

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install clickhouse-server clickhouse-client

# 或使用 Docker
docker pull clickhouse/clickhouse-server:24.8
```

### Q: 权限错误？

**A:** 确保用户有以下权限：
- CREATE TABLE
- INSERT
- SELECT
- DROP TABLE

创建测试用户：

```sql
CREATE USER test_user IDENTIFIED BY 'test_password';
GRANT ALL ON test_db.* TO test_user;
```

### Q: 如何开启调试日志？

**A:** 在 `main.go` 中已经设置了 `Debug: true`，如果想查看更多信息：

```go
Debug: true,
Debugf: func(format string, v ...any) {
    fmt.Printf("[DEBUG] "+format+"\n", v...)
},
```

## 📝 自定义测试

### 测试自己的数据

修改 `testData` 数组：

```go
testData := []struct {
    id          uint32
    name        string
    amountSmall string
    amountMed   string
    amountLarge string
}{
    {
        id:          1,
        name:        "我的测试",
        amountSmall: "123.456",
        amountMed:   "789012345678901234567890.123",
        amountLarge: "最大154位的数字...",
    },
}
```

### 测试不同精度

修改建表 SQL：

```go
createSQL := fmt.Sprintf(`
    CREATE TABLE %s (
        id UInt32,
        test_decimal_77 Decimal(77, 10),   // 最小 Decimal512
        test_decimal_100 Decimal(100, 20), // 中等精度
        test_decimal_154 Decimal(154, 50)  // 最大精度
    ) ENGINE = MergeTree()
    ORDER BY id
`, tableName)
```

## 🎯 测试场景

### 场景 1: 金融计算（高精度货币）

```go
// 加密货币价格，需要超高精度
price := decimal.RequireFromString("0.000000000000000000000123456789")
```

### 场景 2: 科学计算（大数值）

```go
// 天文学计算，需要超大数值
distance := decimal.RequireFromString("9460730472580800000000000000")
```

### 场景 3: 统计分析（极端精度）

```go
// 概率计算，需要超多小数位
probability := decimal.RequireFromString("0.12345678901234567890123456789012345678901234567890")
```

## 📦 部署到生产环境

### 1. 编译生产版本

```bash
cd /root/clickhouse-go/examples/decimal512_demo
GOOS=linux GOARCH=amd64 go build -o decimal512_demo_linux
```

### 2. 配置生产参数

```go
conn, err := clickhouse.Open(&clickhouse.Options{
    Addr: []string{"prod-clickhouse:9000"},
    Auth: clickhouse.Auth{
        Database: os.Getenv("CH_DATABASE"),
        Username: os.Getenv("CH_USERNAME"),
        Password: os.Getenv("CH_PASSWORD"),
    },
    DialTimeout: 10 * time.Second,
    MaxOpenConns: 10,
    MaxIdleConns: 5,
    ConnMaxLifetime: time.Hour,
    Debug: false,  // 生产环境关闭调试
})
```

### 3. 使用环境变量

```bash
export CH_DATABASE=production
export CH_USERNAME=app_user
export CH_PASSWORD=secure_password
./decimal512_demo_linux
```

## 🔗 相关资源

- [完整文档](./README.md)
- [验证报告](../../docs/2025_1026_03_decimal512_verification.md)
- [ClickHouse 文档](https://clickhouse.com/docs)

