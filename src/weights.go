package tango

import (
	"strconv"
	"strings"
)

func parseWeights(data string) ([]float32, error) {
	parts := strings.Split(data, ",")
	weights := make([]float32, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		val, err := strconv.ParseFloat(trimmed, 32)
		if err != nil {
			return nil, err
		}
		weights = append(weights, float32(val))
	}
	return weights, nil
}
