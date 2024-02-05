package flags

import (
	"errors"
	"fmt"
	"strings"
)

type ArrayFlags []string

func (i *ArrayFlags) String() string {
	return fmt.Sprintf("%v", &i)
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *ArrayFlags) Split(delimiter string) (map[string]string, error) {
	m := make(map[string]string)

	for _, e := range *i {
		if !strings.Contains(e, delimiter) {
			return nil, errors.New(fmt.Sprintf("missing delimiter '%s' in pair: '%s'", delimiter, e))
		}

		parts := strings.Split(e, delimiter)
		m[parts[0]] = parts[1]
	}

	return m, nil
}
