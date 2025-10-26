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

func TestStdDecimal512Tuple(t *testing.T) {
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
				CREATE TABLE std_test_decimal512_tuple (
					Col1 Tuple(id UInt32, amount Decimal(100, 25))
				) Engine MergeTree() ORDER BY tuple()
			`
			defer func() {
				conn.Exec("DROP TABLE std_test_decimal512_tuple")
			}()
			_, err = conn.Exec(ddl)
			require.NoError(t, err)

			scope, err := conn.Begin()
			require.NoError(t, err)
			batch, err := scope.Prepare("INSERT INTO std_test_decimal512_tuple")
			require.NoError(t, err)

			val := decimal.RequireFromString("12345678901234567890.1234567890123456789012345")

			// Insert as slice
			_, err = batch.Exec([]any{uint32(123), val})
			require.NoError(t, err)
			require.NoError(t, scope.Commit())

			rows, err := conn.Query("SELECT * FROM std_test_decimal512_tuple")
			require.NoError(t, err)

			for rows.Next() {
				var result []any
				err := rows.Scan(&result)
				require.NoError(t, err)
				require.Len(t, result, 2)

				assert.Equal(t, uint32(123), result[0])
				assert.True(t, val.Equal(result[1].(decimal.Decimal)))
			}
			require.NoError(t, rows.Err())
		})
	}
}

func TestStdDecimal512Map(t *testing.T) {
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
				CREATE TABLE std_test_decimal512_map (
					Col1 Map(String, Decimal(90, 20))
				) Engine MergeTree() ORDER BY tuple()
			`
			defer func() {
				conn.Exec("DROP TABLE std_test_decimal512_map")
			}()
			_, err = conn.Exec(ddl)
			require.NoError(t, err)

			scope, err := conn.Begin()
			require.NoError(t, err)
			batch, err := scope.Prepare("INSERT INTO std_test_decimal512_map")
			require.NoError(t, err)

			inputMap := map[string]decimal.Decimal{
				"price1": decimal.RequireFromString("111111111111111111111111111111.11111111111111111111"),
				"price2": decimal.RequireFromString("222222222222222222222222222222.22222222222222222222"),
			}

			_, err = batch.Exec(inputMap)
			require.NoError(t, err)
			require.NoError(t, scope.Commit())

			rows, err := conn.Query("SELECT * FROM std_test_decimal512_map")
			require.NoError(t, err)

			for rows.Next() {
				var result map[string]decimal.Decimal
				err := rows.Scan(&result)
				require.NoError(t, err)
				require.Len(t, result, 2)

				for key, expected := range inputMap {
					actual, ok := result[key]
					require.True(t, ok, "key %s not found", key)
					assert.True(t, expected.Equal(actual))
				}
			}
			require.NoError(t, rows.Err())
		})
	}
}

func TestStdDecimal512NestedArray(t *testing.T) {
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
				CREATE TABLE std_test_decimal512_nested (
					Col1 Array(Array(Decimal(80, 15)))
				) Engine MergeTree() ORDER BY tuple()
			`
			defer func() {
				conn.Exec("DROP TABLE std_test_decimal512_nested")
			}()
			_, err = conn.Exec(ddl)
			require.NoError(t, err)

			scope, err := conn.Begin()
			require.NoError(t, err)
			batch, err := scope.Prepare("INSERT INTO std_test_decimal512_nested")
			require.NoError(t, err)

			val1 := decimal.RequireFromString("111111111111111111111111111111.111111111111111")
			val2 := decimal.RequireFromString("222222222222222222222222222222.222222222222222")

			nested := [][]decimal.Decimal{
				{val1, val2},
			}

			_, err = batch.Exec(nested)
			require.NoError(t, err)
			require.NoError(t, scope.Commit())

			rows, err := conn.Query("SELECT * FROM std_test_decimal512_nested")
			require.NoError(t, err)

			for rows.Next() {
				var result [][]decimal.Decimal
				err := rows.Scan(&result)
				require.NoError(t, err)
				require.Len(t, result, 1)
				require.Len(t, result[0], 2)

				assert.True(t, val1.Equal(result[0][0]))
				assert.True(t, val2.Equal(result[0][1]))
			}
			require.NoError(t, rows.Err())
		})
	}
}

func TestStdDecimal512ArrayNullable(t *testing.T) {
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
				CREATE TABLE std_test_decimal512_array_null (
					Col1 Array(Nullable(Decimal(100, 25)))
				) Engine MergeTree() ORDER BY tuple()
			`
			defer func() {
				conn.Exec("DROP TABLE std_test_decimal512_array_null")
			}()
			_, err = conn.Exec(ddl)
			require.NoError(t, err)

			scope, err := conn.Begin()
			require.NoError(t, err)
			batch, err := scope.Prepare("INSERT INTO std_test_decimal512_array_null")
			require.NoError(t, err)

			val := decimal.RequireFromString("555555555555555555555555555555.5555555555555555555555555")

			input := []*decimal.Decimal{&val, nil, &val}
			_, err = batch.Exec(input)
			require.NoError(t, err)
			require.NoError(t, scope.Commit())

			rows, err := conn.Query("SELECT * FROM std_test_decimal512_array_null")
			require.NoError(t, err)

			for rows.Next() {
				var result []*decimal.Decimal
				err := rows.Scan(&result)
				require.NoError(t, err)
				require.Len(t, result, 3)

				require.NotNil(t, result[0])
				assert.True(t, val.Equal(*result[0]))
				require.Nil(t, result[1])
				require.NotNil(t, result[2])
				assert.True(t, val.Equal(*result[2]))
			}
			require.NoError(t, rows.Err())
		})
	}
}
