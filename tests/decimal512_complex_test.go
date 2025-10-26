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

// TestDecimal512Tuple tests Tuple containing Decimal512
func TestDecimal512Tuple(t *testing.T) {
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
			CREATE TABLE test_decimal512_tuple (
				Col1 Tuple(id UInt32, amount Decimal(100, 20), name String)
			) Engine MergeTree() ORDER BY tuple()
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512_tuple")
		}()
		require.NoError(t, conn.Exec(ctx, ddl))
		
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_decimal512_tuple")
		require.NoError(t, err)
		
		val := decimal.RequireFromString("12345678901234567890.12345678901234567890")
		
		// Tuple as slice
		tuple := []any{uint32(123), val, "test"}
		require.NoError(t, batch.Append(tuple))
		require.NoError(t, batch.Send())
		
		var result []any
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_decimal512_tuple").Scan(&result))
		require.Len(t, result, 3)
		
		assert.Equal(t, uint32(123), result[0])
		assert.True(t, val.Equal(result[1].(decimal.Decimal)), "expected %v, got %v", val, result[1])
		assert.Equal(t, "test", result[2])
	})
}

// TestDecimal512NamedTuple tests Named Tuple containing Decimal512
func TestDecimal512NamedTuple(t *testing.T) {
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
		
		type Transaction struct {
			ID     uint32          `ch:"id"`
			Amount decimal.Decimal `ch:"amount"`
			Status string          `ch:"status"`
		}
		
		const ddl = `
			CREATE TABLE test_decimal512_named_tuple (
				Col1 Tuple(id UInt32, amount Decimal(120, 30), status String)
			) Engine MergeTree() ORDER BY tuple()
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512_named_tuple")
		}()
		require.NoError(t, conn.Exec(ctx, ddl))
		
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_decimal512_named_tuple")
		require.NoError(t, err)
		
		val := decimal.RequireFromString("999999999999999999999999999999.123456789012345678901234567890")
		input := Transaction{
			ID:     456,
			Amount: val,
			Status: "completed",
		}
		
		require.NoError(t, batch.Append(input))
		require.NoError(t, batch.Send())
		
		var result Transaction
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_decimal512_named_tuple").Scan(&result))
		
		assert.Equal(t, uint32(456), result.ID)
		assert.True(t, val.Equal(result.Amount), "expected %v, got %v", val, result.Amount)
		assert.Equal(t, "completed", result.Status)
	})
}

// TestDecimal512Map tests Map with Decimal512 values
func TestDecimal512Map(t *testing.T) {
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
			CREATE TABLE test_decimal512_map (
				Col1 Map(String, Decimal(90, 25))
			) Engine MergeTree() ORDER BY tuple()
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512_map")
		}()
		require.NoError(t, conn.Exec(ctx, ddl))
		
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_decimal512_map")
		require.NoError(t, err)
		
		inputMap := map[string]decimal.Decimal{
			"price1": decimal.RequireFromString("111111111111111111111111111111.1111111111111111111111111"),
			"price2": decimal.RequireFromString("222222222222222222222222222222.2222222222222222222222222"),
			"price3": decimal.RequireFromString("333333333333333333333333333333.3333333333333333333333333"),
		}
		
		require.NoError(t, batch.Append(inputMap))
		require.NoError(t, batch.Send())
		
		var result map[string]decimal.Decimal
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_decimal512_map").Scan(&result))
		require.Len(t, result, 3)
		
		for key, expected := range inputMap {
			actual, ok := result[key]
			require.True(t, ok, "key %s not found", key)
			assert.True(t, expected.Equal(actual), "key %s: expected %v, got %v", key, expected, actual)
		}
	})
}

// TestDecimal512NestedArray tests nested arrays with Decimal512
func TestDecimal512NestedArray(t *testing.T) {
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
			CREATE TABLE test_decimal512_nested_array (
				Col1 Array(Array(Decimal(80, 15)))
			) Engine MergeTree() ORDER BY tuple()
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512_nested_array")
		}()
		require.NoError(t, conn.Exec(ctx, ddl))
		
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_decimal512_nested_array")
		require.NoError(t, err)
		
		val1 := decimal.RequireFromString("111111111111111111111111111111.111111111111111")
		val2 := decimal.RequireFromString("222222222222222222222222222222.222222222222222")
		val3 := decimal.RequireFromString("333333333333333333333333333333.333333333333333")
		val4 := decimal.RequireFromString("444444444444444444444444444444.444444444444444")
		
		nested := [][]decimal.Decimal{
			{val1, val2},
			{val3, val4},
		}
		
		require.NoError(t, batch.Append(nested))
		require.NoError(t, batch.Send())
		
		var result [][]decimal.Decimal
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_decimal512_nested_array").Scan(&result))
		require.Len(t, result, 2)
		require.Len(t, result[0], 2)
		require.Len(t, result[1], 2)
		
		assert.True(t, val1.Equal(result[0][0]))
		assert.True(t, val2.Equal(result[0][1]))
		assert.True(t, val3.Equal(result[1][0]))
		assert.True(t, val4.Equal(result[1][1]))
	})
}

// TestDecimal512ArrayOfNullable tests Array(Nullable(Decimal512))
func TestDecimal512ArrayOfNullable(t *testing.T) {
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
			CREATE TABLE test_decimal512_array_nullable (
				Col1 Array(Nullable(Decimal(110, 35)))
			) Engine MergeTree() ORDER BY tuple()
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512_array_nullable")
		}()
		require.NoError(t, conn.Exec(ctx, ddl))
		
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_decimal512_array_nullable")
		require.NoError(t, err)
		
		val1 := decimal.RequireFromString("555555555555555555555555555555.55555555555555555555555555555555555")
		val2 := decimal.RequireFromString("666666666666666666666666666666.66666666666666666666666666666666666")
		
		input := []*decimal.Decimal{&val1, nil, &val2, nil}
		require.NoError(t, batch.Append(input))
		require.NoError(t, batch.Send())
		
		var result []*decimal.Decimal
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_decimal512_array_nullable").Scan(&result))
		require.Len(t, result, 4)
		
		require.NotNil(t, result[0])
		assert.True(t, val1.Equal(*result[0]))
		require.Nil(t, result[1])
		require.NotNil(t, result[2])
		assert.True(t, val2.Equal(*result[2]))
		require.Nil(t, result[3])
	})
}

// TestDecimal512ComplexMixed tests complex combinations
func TestDecimal512ComplexMixed(t *testing.T) {
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
		
		// Complex type: Tuple of (Array of Nullable Decimal512, Map)
		const ddl = `
			CREATE TABLE test_decimal512_complex (
				Col1 Tuple(
					amounts Array(Nullable(Decimal(95, 20))),
					prices Map(String, Decimal(85, 18))
				)
			) Engine MergeTree() ORDER BY tuple()
		`
		defer func() {
			conn.Exec(ctx, "DROP TABLE IF EXISTS test_decimal512_complex")
		}()
		require.NoError(t, conn.Exec(ctx, ddl))
		
		batch, err := conn.PrepareBatch(ctx, "INSERT INTO test_decimal512_complex")
		require.NoError(t, err)
		
		val1 := decimal.RequireFromString("77777777777777777777777777777777777777777777777777777777.77777777777777777777")
		val2 := decimal.RequireFromString("88888888888888888888888888888888888888888888888888888888.88888888888888888888")
		price1 := decimal.RequireFromString("99999999999999999999999999999999999999999999999999999999.999999999999999999")
		price2 := decimal.RequireFromString("11111111111111111111111111111111111111111111111111111111.111111111111111111")
		
		type ComplexData struct {
			Amounts []*decimal.Decimal         `ch:"amounts"`
			Prices  map[string]decimal.Decimal `ch:"prices"`
		}
		
		input := ComplexData{
			Amounts: []*decimal.Decimal{&val1, nil, &val2},
			Prices: map[string]decimal.Decimal{
				"item1": price1,
				"item2": price2,
			},
		}
		
		require.NoError(t, batch.Append(input))
		require.NoError(t, batch.Send())
		
		var result ComplexData
		require.NoError(t, conn.QueryRow(ctx, "SELECT * FROM test_decimal512_complex").Scan(&result))
		
		require.Len(t, result.Amounts, 3)
		require.NotNil(t, result.Amounts[0])
		assert.True(t, val1.Equal(*result.Amounts[0]))
		require.Nil(t, result.Amounts[1])
		require.NotNil(t, result.Amounts[2])
		assert.True(t, val2.Equal(*result.Amounts[2]))
		
		require.Len(t, result.Prices, 2)
		assert.True(t, price1.Equal(result.Prices["item1"]))
		assert.True(t, price2.Equal(result.Prices["item2"]))
	})
}

