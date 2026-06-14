package cli

import (
	"errors"

	"github.com/tamnd/reuters-cli/reuters"
)

func isUnknownSection(err error) bool {
	return errors.Is(err, reuters.ErrUnknownSection)
}
