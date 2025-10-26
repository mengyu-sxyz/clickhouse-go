package std

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdDecimal512(t *testing.T) {
	dsns := map[string]clickhouse.Protocol{"Native": clickhouse.Native, "Http": clickhouse.HTTP}
	useSSL, err := strconv.ParseBool(clickhouse_tests.GetEnv("CLICKHOUSE_USE_SSL", "false"))
	require.NoError(t, err)
	for name, protocol := range dsns {
		t.Run(fmt.Sprintf("%s Protocol", name), func(t *testing.T) {
			conn, err := GetStdDSNConnection(protocol, useSSL, nil)
			require.NoError(t, err)
			if !CheckMinServerVersion(conn, 24, 8, 0) {
				t.Skip(fmt.Errorf("Decimal512 requires ClickHouse 24.8+"))
				return
			}
			const ddl = `
			CREATE TABLE std_test_decimal512 (
				Col1 Decimal(80, 10)
				, Col2 Decimal(120, 20)
				, Col3 Nullable(Decimal(100, 15))
				, Col4 Array(Decimal(90, 12))
			) Engine MergeTree() ORDER BY tuple()
		`
			defer func() {
				conn.Exec("DROP TABLE std_test_decimal512")
			}()
			_, err = conn.Exec(ddl)
			require.NoError(t, err)
			scope, err := conn.Begin()
			require.NoError(t, err)
			batch, err := scope.Prepare("INSERT INTO std_test_decimal512")
			require.NoError(t, err)
			
			val1 := decimal.RequireFromString("12345678901234567890.1234567890")
			val2 := decimal.RequireFromString("98765432109876543210.98765432109876543210")
			val3 := decimal.RequireFromString("55555555555555555555.555555555555555")
			arrayVals := []decimal.Decimal{
				decimal.RequireFromString("111.222333444555"),
				decimal.RequireFromString("666.777888999000"),
			}
			
			_, err = batch.Exec(val1, val2, val3, arrayVals)
			require.NoError(t, err)
			require.NoError(t, scope.Commit())
			
			var (
				col1 decimal.Decimal
				col2 decimal.Decimal
				col3 decimal.Decimal
				col4 []decimal.Decimal
			)
			rows, err := conn.Query("SELECT * FROM std_test_decimal512")
			require.NoError(t, err)
			columnTypes, err := rows.ColumnTypes()
			require.NoError(t, err)
			
			// Verify column metadata
			for i, column := range columnTypes {
				switch i {
				case 0:
					nullable, nullableOk := column.Nullable()
					assert.False(t, nullable)
					assert.True(t, nullableOk)

					precision, scale, ok := column.DecimalSize()
					assert.Equal(t, int64(10), scale)
					assert.Equal(t, int64(80), precision)
					assert.True(t, ok)
				case 1:
					nullable, nullableOk := column.Nullable()
					assert.False(t, nullable)
					assert.True(t, nullableOk)

					precision, scale, ok := column.DecimalSize()
					assert.Equal(t, int64(20), scale)
					assert.Equal(t, int64(120), precision)
					assert.True(t, ok)
				case 2:
					nullable, nullableOk := column.Nullable()
					assert.True(t, nullable)
					assert.True(t, nullableOk)

					precision, scale, ok := column.DecimalSize()
					assert.Equal(t, int64(15), scale)
					assert.Equal(t, int64(100), precision)
					assert.True(t, ok)
				case 3:
					nullable, nullableOk := column.Nullable()
					assert.False(t, nullable)
					assert.True(t, nullableOk)
				}
			}
			
			for rows.Next() {
				err := rows.Scan(&col1, &col2, &col3, &col4)
				require.NoError(t, err)
				
				assert.True(t, val1.Equal(col1), "Col1: expected %v, got %v", val1, col1)
				assert.True(t, val2.Equal(col2), "Col2: expected %v, got %v", val2, col2)
				assert.True(t, val3.Equal(col3), "Col3: expected %v, got %v", val3, col3)
				require.Len(t, col4, 2)
				assert.True(t, arrayVals[0].Equal(col4[0]), "Col4[0]: expected %v, got %v", arrayVals[0], col4[0])
				assert.True(t, arrayVals[1].Equal(col4[1]), "Col4[1]: expected %v, got %v", arrayVals[1], col4[1])
			}
			require.NoError(t, rows.Err())
		})
	}
}

func TestStdDecimal512Nullable(t *testing.T) {
	dsns := map[string]clickhouse.Protocol{"Native": clickhouse.Native, "Http": clickhouse.HTTP}
	useSSL, err := strconv.ParseBool(clickhouse_tests.GetEnv("CLICKHOUSE_USE_SSL", "false"))
	require.NoError(t, err)
	for name, protocol := range dsns {
		t.Run(fmt.Sprintf("%s Protocol", name), func(t *testing.T) {
			conn, err := GetStdDSNConnection(protocol, useSSL, nil)
			require.NoError(t, err)
			if !CheckMinServerVersion(conn, 24, 8, 0) {
				t.Skip(fmt.Errorf("Decimal512 requires ClickHouse 24.8+"))
				return
			}
			const ddl = `
			CREATE TABLE std_test_decimal512_nullable (
				Col1 Nullable(Decimal(100, 20))
			) Engine MergeTree() ORDER BY tuple()
		`
			defer func() {
				conn.Exec("DROP TABLE std_test_decimal512_nullable")
			}()
			_, err = conn.Exec(ddl)
			require.NoError(t, err)
			
			scope, err := conn.Begin()
			require.NoError(t, err)
			batch, err := scope.Prepare("INSERT INTO std_test_decimal512_nullable")
			require.NoError(t, err)
			
			val := decimal.RequireFromString("123456789.98765432101234567890")
			
			// Insert non-null value
			_, err = batch.Exec(val)
			require.NoError(t, err)
			
			// Insert null value
			_, err = batch.Exec(nil)
			require.NoError(t, err)
			
			require.NoError(t, scope.Commit())
			
			rows, err := conn.Query("SELECT * FROM std_test_decimal512_nullable ORDER BY Col1 NULLS LAST")
			require.NoError(t, err)
			
			rowCount := 0
			for rows.Next() {
				var col1 *decimal.Decimal
				err := rows.Scan(&col1)
				require.NoError(t, err)
				rowCount++
				
				switch rowCount {
				case 1:
					require.NotNil(t, col1)
					assert.True(t, val.Equal(*col1), "expected %v, got %v", val, *col1)
				case 2:
					require.Nil(t, col1)
				}
			}
			assert.Equal(t, 2, rowCount)
			require.NoError(t, rows.Err())
		})
	}
}

func TestStdDecimal512StringInput(t *testing.T) {
	dsns := map[string]clickhouse.Protocol{"Native": clickhouse.Native, "Http": clickhouse.HTTP}
	useSSL, err := strconv.ParseBool(clickhouse_tests.GetEnv("CLICKHOUSE_USE_SSL", "false"))
	require.NoError(t, err)
	for name, protocol := range dsns {
		t.Run(fmt.Sprintf("%s Protocol", name), func(t *testing.T) {
			conn, err := GetStdDSNConnection(protocol, useSSL, nil)
			require.NoError(t, err)
			if !CheckMinServerVersion(conn, 24, 8, 0) {
				t.Skip(fmt.Errorf("Decimal512 requires ClickHouse 24.8+"))
				return
			}
			const ddl = `
			CREATE TABLE std_test_decimal512_string (
				Col1 Decimal(85, 25)
			) Engine MergeTree() ORDER BY tuple()
		`
			defer func() {
				conn.Exec("DROP TABLE std_test_decimal512_string")
			}()
			_, err = conn.Exec(ddl)
			require.NoError(t, err)
			
			scope, err := conn.Begin()
			require.NoError(t, err)
			batch, err := scope.Prepare("INSERT INTO std_test_decimal512_string")
			require.NoError(t, err)
			
			// Test with string input
			strValue := "9876543210.9876543210987654321012345"
			_, err = batch.Exec(strValue)
			require.NoError(t, err)
			require.NoError(t, scope.Commit())
			
			var result decimal.Decimal
			rows, err := conn.Query("SELECT * FROM std_test_decimal512_string")
			require.NoError(t, err)
			
			for rows.Next() {
				err := rows.Scan(&result)
				require.NoError(t, err)
			}
			require.NoError(t, rows.Err())
			
			expected := decimal.RequireFromString(strValue)
			assert.True(t, expected.Equal(result), "expected %v, got %v", expected, result)
		})
	}
}

