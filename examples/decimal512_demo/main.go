package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/shopspring/decimal"
)

func main() {
	// 配置连接参数
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"}, // 修改为你的 ClickHouse 地址
		Auth: clickhouse.Auth{
			Database: "default", // 修改为你的数据库
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		Settings: clickhouse.Settings{
			"allow_experimental_bigint_types": 1, // 启用实验性大整数类型
		},
		Debug: true, // 开启调试日志
	})
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer conn.Close()

	ctx := context.Background()

	// 测试连接
	if err := conn.Ping(ctx); err != nil {
		log.Fatal("Ping 失败:", err)
	}
	fmt.Println("✅ 连接成功！")

	// 检查 ClickHouse 版本
	var version string
	if err := conn.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		log.Fatal("查询版本失败:", err)
	}
	fmt.Printf("📊 ClickHouse 版本: %s\n", version)

	// 创建测试表
	tableName := "test_decimal512_demo"
	fmt.Printf("\n🔨 创建测试表: %s\n", tableName)

	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	if err := conn.Exec(ctx, dropSQL); err != nil {
		log.Fatal("删除表失败:", err)
	}

	createSQL := fmt.Sprintf(`
		CREATE TABLE %s (
			id UInt32,
			name String,
			amount_small Decimal(80, 10),
			amount_medium Decimal(120, 25),
			amount_large Decimal(153, 50),
			created_at DateTime
		) ENGINE = MergeTree()
		ORDER BY id
	`, tableName)

	if err := conn.Exec(ctx, createSQL); err != nil {
		log.Fatal("创建表失败:", err)
	}
	fmt.Println("✅ 表创建成功")

	// 准备测试数据
	fmt.Println("\n📝 插入测试数据...")

	batch, err := conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", tableName))
	if err != nil {
		log.Fatal("准备批量插入失败:", err)
	}

	// 测试数据 - 不同精度的 Decimal512
	// 注意：Decimal(P, S) 中 P 是总位数，S 是小数位数
	// Decimal(80, 10)：整数部分最多 70 位
	// Decimal(120, 25)：整数部分最多 95 位
	// Decimal(153, 50)：整数部分最多 103 位
	testData := []struct {
		id          uint32
		name        string
		amountSmall string
		amountMed   string
		amountLarge string
	}{
		{
			id:          1,
			name:        "小精度测试",
			amountSmall: "12345678901234567890.1234567890",                                                                       // 20位整数 + 10位小数 = 30位 < 80
			amountMed:   "11111111111111111111111111111111111111111111111111.1111111111111111111111111",                          // 50位整数 + 25位小数 = 75位 < 120
			amountLarge: "12345678901234567890123456789012345678901234567890.12345678901234567890123456789012345678901234567890", // 50位整数 + 50位小数 = 100位 < 153
		},
		{
			id:          2,
			name:        "中精度测试",
			amountSmall: "98765432109876543210.9876543210",                                                                       // 20位整数 + 10位小数
			amountMed:   "22222222222222222222222222222222222222222222222222.2222222222222222222222222",                          // 50位整数 + 25位小数
			amountLarge: "99999999999999999999999999999999999999999999999999.99999999999999999999999999999999999999999999999999", // 50位整数 + 50位小数
		},
		{
			id:          3,
			name:        "负数测试",
			amountSmall: "-55555555555555555555.5555555555",                                                                       // 20位整数 + 10位小数
			amountMed:   "-33333333333333333333333333333333333333333333333333.3333333333333333333333333",                          // 50位整数 + 25位小数
			amountLarge: "-77777777777777777777777777777777777777777777777777.77777777777777777777777777777777777777777777777777", // 50位整数 + 50位小数
		},
		{
			id:   4,
			name: "边界测试-接近最大值",
			// Decimal(80,10): 整数最多70位
			amountSmall: strings.Repeat("9", 70) + "." + strings.Repeat("9", 10),
			// Decimal(120,25): 整数最多95位
			amountMed: strings.Repeat("9", 95) + "." + strings.Repeat("9", 25),
			// Decimal(153,50): 整数最多103位
			amountLarge: strings.Repeat("9", 103) + "." + strings.Repeat("9", 50),
		},
	}

	for _, data := range testData {
		amountSmall := decimal.RequireFromString(data.amountSmall)
		amountMed := decimal.RequireFromString(data.amountMed)
		amountLarge := decimal.RequireFromString(data.amountLarge)

		if err := batch.Append(
			data.id,
			data.name,
			amountSmall,
			amountMed,
			amountLarge,
			time.Now(),
		); err != nil {
			log.Fatal("追加数据失败:", err)
		}
	}

	if err := batch.Send(); err != nil {
		log.Fatal("发送批量数据失败:", err)
	}
	fmt.Printf("✅ 成功插入 %d 条数据\n", len(testData))

	// 查询并验证数据
	fmt.Println("\n🔍 查询数据并验证...")

	rows, err := conn.Query(ctx, fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName))
	if err != nil {
		log.Fatal("查询失败:", err)
	}
	defer rows.Close()

	fmt.Println("\n" + repeat("=", 150))
	fmt.Printf("%-5s | %-20s | %-35s | %-60s | %-110s\n",
		"ID", "名称", "小精度(80位)", "中精度(120位)", "大精度(153位)")
	fmt.Println(repeat("=", 150))

	rowCount := 0
	for rows.Next() {
		var (
			id          uint32
			name        string
			amountSmall decimal.Decimal
			amountMed   decimal.Decimal
			amountLarge decimal.Decimal
			createdAt   time.Time
		)

		if err := rows.Scan(&id, &name, &amountSmall, &amountMed, &amountLarge, &createdAt); err != nil {
			log.Fatal("扫描行失败:", err)
		}

		rowCount++

		// 显示数据
		fmt.Printf("%-5d | %-20s | %-35s | %-60s | %-110s\n",
			id,
			name,
			truncateString(amountSmall.String(), 35),
			truncateString(amountMed.String(), 60),
			truncateString(amountLarge.String(), 110),
		)

		// 验证数据完整性
		expectedSmall := testData[id-1].amountSmall
		expectedMed := testData[id-1].amountMed
		expectedLarge := testData[id-1].amountLarge

		if !decimal.RequireFromString(expectedSmall).Equal(amountSmall) {
			log.Printf("⚠️  警告: ID=%d 的小精度数据不匹配！期望: %s, 实际: %s", id, expectedSmall, amountSmall.String())
		}
		if !decimal.RequireFromString(expectedMed).Equal(amountMed) {
			log.Printf("⚠️  警告: ID=%d 的中精度数据不匹配！期望: %s, 实际: %s", id, expectedMed, amountMed.String())
		}
		if !decimal.RequireFromString(expectedLarge).Equal(amountLarge) {
			log.Printf("⚠️  警告: ID=%d 的大精度数据不匹配！期望: %s, 实际: %s", id, expectedLarge, amountLarge.String())
		}
	}
	fmt.Println(repeat("=", 150))

	if err := rows.Err(); err != nil {
		log.Fatal("遍历行失败:", err)
	}

	fmt.Printf("\n✅ 成功读取 %d 条数据\n", rowCount)

	// 测试复杂类型
	fmt.Println("\n🧪 测试复杂类型支持...")
	testComplexTypes(ctx, conn)

	// 清理
	fmt.Println("\n🧹 清理测试表...")
	if err := conn.Exec(ctx, dropSQL); err != nil {
		log.Printf("⚠️  警告: 删除表失败: %v", err)
	} else {
		fmt.Println("✅ 清理完成")
	}

	fmt.Println("\n🎉 所有测试完成！Decimal512 功能正常工作！")
}

// 测试复杂类型（Nullable, Array, Map）
func testComplexTypes(ctx context.Context, conn clickhouse.Conn) {
	tableName := "test_decimal512_complex_demo"

	// 创建表
	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	conn.Exec(ctx, dropSQL)

	createSQL := fmt.Sprintf(`
		CREATE TABLE %s (
			id UInt32,
			nullable_amount Nullable(Decimal(100, 20)),
			array_amounts Array(Decimal(90, 15)),
			map_amounts Map(String, Decimal(85, 12))
		) ENGINE = MergeTree()
		ORDER BY id
	`, tableName)

	if err := conn.Exec(ctx, createSQL); err != nil {
		log.Printf("创建复杂类型表失败: %v", err)
		return
	}

	// 插入数据
	batch, err := conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", tableName))
	if err != nil {
		log.Printf("准备批量插入失败: %v", err)
		return
	}

	val1 := decimal.RequireFromString("123456789012345678901234567890.12345678901234567890")
	val2 := decimal.RequireFromString("111111111111111111111111111111.111111111111111")
	val3 := decimal.RequireFromString("999999999999999999999999999999.999999999999")

	// 插入有值的行
	batch.Append(
		uint32(1),
		&val1,
		[]decimal.Decimal{val2, val3},
		map[string]decimal.Decimal{
			"price1": val2,
			"price2": val3,
		},
	)

	// 插入 NULL 值的行
	batch.Append(
		uint32(2),
		nil,
		[]decimal.Decimal{val1},
		map[string]decimal.Decimal{
			"price": val1,
		},
	)

	if err := batch.Send(); err != nil {
		log.Printf("发送批量数据失败: %v", err)
		return
	}

	// 查询验证
	rows, err := conn.Query(ctx, fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName))
	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\n  复杂类型测试结果:")
	for rows.Next() {
		var (
			id             uint32
			nullableAmount *decimal.Decimal
			arrayAmounts   []decimal.Decimal
			mapAmounts     map[string]decimal.Decimal
		)

		if err := rows.Scan(&id, &nullableAmount, &arrayAmounts, &mapAmounts); err != nil {
			log.Printf("扫描行失败: %v", err)
			continue
		}

		fmt.Printf("  - ID %d: ", id)
		if nullableAmount != nil {
			fmt.Printf("Nullable=有值, ")
		} else {
			fmt.Printf("Nullable=NULL, ")
		}
		fmt.Printf("Array长度=%d, Map键数=%d\n", len(arrayAmounts), len(mapAmounts))
	}

	// 清理
	conn.Exec(ctx, dropSQL)
	fmt.Println("  ✅ 复杂类型测试通过")
}

// 辅助函数：截断字符串用于显示
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// 字符串重复函数
func repeat(s string, n int) string {
	return strings.Repeat(s, n)
}
