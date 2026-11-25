package main

import "errors"

// errNoPublishTime reports a version whose publish time could not be found.
var errNoPublishTime = errors.New("no publish time found")
