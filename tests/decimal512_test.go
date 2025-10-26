package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecimal512(t *testing.T) {
	TestProtocols(t, func(t *testing.T, protocol clickhouse.Protocol) {
		conn, err := GetNativeConnection(t, protocol, clickhouse.Settings{
			"allow_experimental_bigint_types": 1,
		}, nil, &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		})
		require.NoError(t, err)
		ctx := context.Background()
		if !CheckMinServerServerVersion(conn, 24, 8, 0) {
			t.Skip(fmt.Errorf("Decimal512 requires ClickHouse 24.8+"))
			return
		}
		const ddl = `
			CREATE TABLE test_decimal512 (
				  Col1 Decimal(77, 10)
				, Col2 Decimal(100, 20)
				, Col3 Decimal(154, 30)
			) Engine MergeTree() ORDER BY tuple()
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512")
		}()
		require.NoError(t, conn.Exec(ctx, ddl))
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_decimal512")
		require.NoError(t, err)
		
		// Test with various decimal values
		val1 := decimal.RequireFromString("123456789012345678901234567890.1234567890")
		val2 := decimal.RequireFromString("987654321098765432109876543210.98765432109876543210")
		val3 := decimal.RequireFromString("111111111111111111111111111111.111111111111111111111111111111")
		
		require.NoError(t, batch.Append(val1, val2, val3))
		require.Equal(t, 1, batch.Rows())
		require.NoError(t, batch.Send())
		
		var (
			col1 decimal.Decimal
			col2 decimal.Decimal
			col3 decimal.Decimal
		)
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_decimal512").Scan(&col1, &col2, &col3))
		
		// Compare values with appropriate precision
		assert.True(t, val1.Equal(col1), "Col1: expected %v, got %v", val1, col1)
		assert.True(t, val2.Equal(col2), "Col2: expected %v, got %v", val2, col2)
		assert.True(t, val3.Equal(col3), "Col3: expected %v, got %v", val3, col3)
	})
}

func TestNegativeDecimal512(t *testing.T) {
	TestProtocols(t, func(t *testing.T, protocol clickhouse.Protocol) {
		conn, err := GetNativeConnection(t, protocol, clickhouse.Settings{
			"allow_experimental_bigint_types": 1,
		}, nil, &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		})
		ctx := context.Background()
		require.NoError(t, err)
		if !CheckMinServerServerVersion(conn, 24, 8, 0) {
			t.Skip(fmt.Errorf("Decimal512 requires ClickHouse 24.8+"))
			return
		}
		require.NoError(t, conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512_negative"))
		const ddl = `
		CREATE TABLE test_decimal512_negative (
			  Col1 Nullable(Decimal(80, 10)),
			  Col2 Nullable(Decimal(120, 25)),
			  Col3 Nullable(Decimal(154, 40))
		) Engine MergeTree() ORDER BY tuple()
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512_negative")
		}()
		require.NoError(t, conn.Exec(ctx, ddl))
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_decimal512_negative")
		require.NoError(t, err)
		
		// Test with negative and null values
		val1 := decimal.RequireFromString("-123456789012345678901234567890.1234567890")
		val2 := decimal.RequireFromString("-987654321098765432109876543210.98765432109876543210")
		
		require.NoError(t, batch.Append(val1, val2, nil))
		require.NoError(t, batch.Append(nil, val2, val1))
		require.NoError(t, batch.Send())
		
		rows, err := conn.Query(ctx, "SELECT * FROM test_decimal512_negative ORDER BY Col1 NULLS LAST")
		require.NoError(t, err)
		defer rows.Close()
		
		var rowCount int
		for rows.Next() {
			var (
				col1 *decimal.Decimal
				col2 *decimal.Decimal
				col3 *decimal.Decimal
			)
			require.NoError(t, rows.Scan(&col1, &col2, &col3))
			rowCount++
			
			switch rowCount {
			case 1:
				require.Nil(t, col1)
				require.NotNil(t, col2)
				require.NotNil(t, col3)
				assert.True(t, val2.Equal(*col2))
				assert.True(t, val1.Equal(*col3))
			case 2:
				require.NotNil(t, col1)
				require.NotNil(t, col2)
				require.Nil(t, col3)
				assert.True(t, val1.Equal(*col1))
				assert.True(t, val2.Equal(*col2))
			}
		}
		assert.Equal(t, 2, rowCount)
	})
}

func TestDecimal512Array(t *testing.T) {
	TestProtocols(t, func(t *testing.T, protocol clickhouse.Protocol) {
		conn, err := GetNativeConnection(t, protocol, clickhouse.Settings{
			"allow_experimental_bigint_types": 1,
		}, nil, &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		})
		require.NoError(t, err)
		ctx := context.Background()
		if !CheckMinServerServerVersion(conn, 24, 8, 0) {
			t.Skip(fmt.Errorf("Decimal512 requires ClickHouse 24.8+"))
			return
		}
		const ddl = `
			CREATE TABLE test_decimal512_array (
				Col1 Array(Decimal(100, 10))
			) Engine MergeTree() ORDER BY tuple()
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512_array")
		}()
		require.NoError(t, conn.Exec(ctx, ddl))
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_decimal512_array")
		require.NoError(t, err)
		
		values := []decimal.Decimal{
			decimal.RequireFromString("123.456"),
			decimal.RequireFromString("789.012"),
			decimal.RequireFromString("-345.678"),
		}
		
		require.NoError(t, batch.Append(values))
		require.NoError(t, batch.Send())
		
		var result []decimal.Decimal
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_decimal512_array").Scan(&result))
		require.Len(t, result, 3)
		
		for i, expected := range values {
			assert.True(t, expected.Equal(result[i]), "Index %d: expected %v, got %v", i, expected, result[i])
		}
	})
}

func TestDecimal512String(t *testing.T) {
	TestProtocols(t, func(t *testing.T, protocol clickhouse.Protocol) {
		conn, err := GetNativeConnection(t, protocol, clickhouse.Settings{
			"allow_experimental_bigint_types": 1,
		}, nil, &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		})
		require.NoError(t, err)
		ctx := context.Background()
		if !CheckMinServerServerVersion(conn, 24, 8, 0) {
			t.Skip(fmt.Errorf("Decimal512 requires ClickHouse 24.8+"))
			return
		}
		const ddl = `
			CREATE TABLE test_decimal512_string (
				Col1 Decimal(90, 15)
			) Engine MergeTree() ORDER BY tuple()
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512_string")
		}()
		require.NoError(t, conn.Exec(ctx, ddl))
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_decimal512_string")
		require.NoError(t, err)
		
		// Test string input
		strValue := "123456789012345678901234567890.123456789012345"
		require.NoError(t, batch.AppendRow(strValue))
		require.NoError(t, batch.Send())
		
		var result decimal.Decimal
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_decimal512_string").Scan(&result))
		
		expected := decimal.RequireFromString(strValue)
		assert.True(t, expected.Equal(result), "expected %v, got %v", expected, result)
	})
}

