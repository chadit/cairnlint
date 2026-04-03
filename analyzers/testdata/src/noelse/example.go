package noelse

import "errors"

// BadIfElse uses an else branch where an early return would be clearer.
func BadIfElse(val int) string {
	if val > 0 {
		return "positive"
	} else { // want `rewrite if-else as early return or guard clause`
		return "non-positive"
	}
}

// BadIfElseIf chains else-if blocks.
func BadIfElseIf(val int) string {
	if val > 0 {
		return "positive"
	} else if val == 0 { // want `rewrite if-else as early return or guard clause`
		return "zero"
	} else { // want `rewrite if-else as early return or guard clause`
		return "negative"
	}
}

// GoodEarlyReturn uses a guard clause with no else.
func GoodEarlyReturn(val int) (string, error) {
	if val < 0 {
		return "", errors.New("negative value")
	}

	return "ok", nil
}

// GoodIfWithoutElse is fine because there is no else branch.
func GoodIfWithoutElse(val int) {
	if val > 0 {
		_ = "positive"
	}
}
