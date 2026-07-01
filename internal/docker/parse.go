package docker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

// ParseInspect parses the JSON array emitted by `docker inspect`.
func ParseInspect(b []byte) ([]Container, error) {
	var cs []Container
	if err := json.Unmarshal(b, &cs); err != nil {
		return nil, fmt.Errorf("parse inspect: %w", err)
	}
	return cs, nil
}

// ParsePS parses newline-delimited JSON objects from
// `docker ps --format '{{json .}}'`.
func ParsePS(b []byte) ([]ContainerSummary, error) {
	var out []ContainerSummary
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var s ContainerSummary
		if err := json.Unmarshal(line, &s); err != nil {
			return nil, fmt.Errorf("parse ps line: %w", err)
		}
		out = append(out, s)
	}
	return out, sc.Err()
}
