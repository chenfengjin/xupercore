package utils

import (
	"errors"
	"regexp"
	"strings"
)

func IsAccount(name string) int {
	if name == "" {
		return -1
	}
	if !strings.HasPrefix(name, GetAccountPrefix()) {
		return 0
	}
	// error means compile error,ignore it as it is sure to compile success
	matched,_:=regexp.MatchString("XC\\d16@[a-z|A-Z]]+",name)

	if matched{
		return 0
	}
	return -1
}


// ValidRawAccount validate account number
func ValidRawAccount(accountName string) error {
	matched, _ := regexp.MatchString("\\d16", accountName)
	if !matched {
		return errors.New("invalid account")
	}
	return nil
}
