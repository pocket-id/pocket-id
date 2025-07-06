package sqlite

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	sqlitelib "github.com/glebarez/go-sqlite"
	"golang.org/x/text/unicode/norm"
)

func RegisterSqliteFunctions() {
	// Register the `normalize(text, form)` function, which performs Unicode normalization on the text
	// This is currently only used in migration functions
	sqlitelib.MustRegisterDeterministicScalarFunction("normalize", 2, func(ctx *sqlitelib.FunctionContext, args []driver.Value) (driver.Value, error) {
		if len(args) != 2 {
			return nil, errors.New("normalize requires 2 arguments")
		}

		arg0, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("first argument for normalize is not a string: %T", args[0])
		}

		arg1, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("second argument for normalize is not a string: %T", args[1])
		}

		var form norm.Form
		switch strings.ToLower(arg1) {
		case "nfc":
			form = norm.NFC
		case "nfd":
			form = norm.NFD
		case "nfkc":
			form = norm.NFKC
		case "nfkd":
			form = norm.NFKD
		default:
			return nil, fmt.Errorf("unsupported form: %s", arg1)
		}

		if len(arg0) == 0 {
			return arg0, nil
		}

		return form.String(arg0), nil
	})
}
