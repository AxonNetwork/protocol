package nodegit

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/Conscience/protocol/util"
)

type MaybeProgress struct {
	Fetched int64
	ToFetch int64
	Error   error
}

func ParseProgress(reader io.Reader, ch chan MaybeProgress) error {
	scanner := bufio.NewScanner(reader)
	scanner.Split(util.ScanCarriageReturns) // lines are broken by \r, not \n

	for scanner.Scan() {
		line := scanner.Text()

		startIdx := strings.Index(line, "Progress: ")
		if startIdx == -1 {
			continue
		}
		line = line[startIdx+10:]

		slashIdx := strings.Index(line, "/")
		if slashIdx == -1 {
			continue
		}

		endIdx := strings.Index(line, " ")
		if endIdx == -1 {
			endIdx = len(line)
		}

		fetchedStr := line[:slashIdx]
		toFetchStr := line[slashIdx+1 : endIdx]

		fetched, err := strconv.ParseInt(fetchedStr, 10, 64)
		if err != nil {
			continue
		}

		toFetch, err := strconv.ParseInt(toFetchStr, 10, 64)
		if err != nil {
			continue
		}

		ch <- MaybeProgress{
			Fetched: fetched,
			ToFetch: toFetch,
		}

		if fetched == toFetch {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
