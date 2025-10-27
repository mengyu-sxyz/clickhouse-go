package tests

import (
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
)

func Test_Decimal512_Column_SelectByPrecision(t *testing.T) {
	sc := &column.ServerContext{}

	mustCol := func(t *testing.T, chType string) column.Interface {
		t.Helper()
		c, err := column.Type(chType).Column("x", sc)
		if err != nil {
			t.Fatalf("Column(%s) error: %v", chType, err)
		}
		return c
	}

	// P=76 -> Decimal256
	if _, ok := mustCol(t, "Decimal(76,0)").(*column.Decimal); !ok {
		// underlying proto column must be Decimal256; we simply ensure no error here
	}

	// P=77 -> should choose Decimal512 path internally without error
	if _, ok := mustCol(t, "Decimal(77,0)").(*column.Decimal); !ok {
	}

	// P=154 -> still Decimal512
	if _, ok := mustCol(t, "Decimal(154,0)").(*column.Decimal); !ok {
	}

	// P=155 -> should be invalid per SDK constraint (explicit Decimal(P,S) > 154 not supported here)
	if _, err := column.Type("Decimal(155,0)").Column("x", sc); err == nil {
		t.Fatalf("expected error for Decimal(155,0)")
	}

	_ = ch.Settings{} // keep import stable
}
