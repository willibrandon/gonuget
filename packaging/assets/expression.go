package assets

import (
	"strings"
)

// PatternExpression is a compiled pattern with optimized matching.
// Reference: ContentModel/Infrastructure/Parser.cs PatternExpression
type PatternExpression struct {
	segments []Segment
	defaults map[string]interface{}
	table    *PatternTable
}

// Segment represents a pattern segment (literal or token).
type Segment interface {
	// TryMatch attempts to match this segment against path.
	// Returns true and end index if match succeeds.
	TryMatch(item **ContentItem, path string, properties map[string]*PropertyDefinition, startIndex int) (int, bool)
}

// LiteralSegment matches exact text.
type LiteralSegment struct {
	text string
}

// TokenSegment matches a property placeholder.
type TokenSegment struct {
	name             string
	delimiter        byte
	matchOnly        bool
	table            *PatternTable
	preserveRawValue bool
}

// NewPatternExpression compiles a pattern definition into an expression.
// Reference: PatternExpression constructor
func NewPatternExpression(pattern *PatternDefinition) *PatternExpression {
	expr := &PatternExpression{
		table:    pattern.Table,
		defaults: make(map[string]interface{}),
	}

	// Copy defaults
	for k, v := range pattern.Defaults {
		expr.defaults[k] = v
	}

	// Parse pattern into segments
	expr.initialize(pattern.Pattern, pattern.PreserveRawValues)

	return expr
}

// initialize parses pattern string into literal and token segments.
// Reference: PatternExpression.Initialize
func (pe *PatternExpression) initialize(pattern string, preserveRawValues bool) {
	scanIndex := 0

	for scanIndex < len(pattern) {
		// Find next token
		beginToken := len(pattern)
		endToken := len(pattern)

		for i := scanIndex; i < len(pattern); i++ {
			ch := pattern[i]
			if beginToken == len(pattern) {
				if ch == '{' {
					beginToken = i
				}
			} else if ch == '}' {
				endToken = i
				break
			}
		}

		// Add literal segment if any
		if scanIndex != beginToken {
			pe.segments = append(pe.segments, &LiteralSegment{
				text: pattern[scanIndex:beginToken],
			})
		}

		// Add token segment if any
		if beginToken != endToken {
			var delimiter byte
			if endToken+1 < len(pattern) {
				delimiter = pattern[endToken+1]
			}

			matchOnly := pattern[endToken-1] == '?'

			beginName := beginToken + 1
			endName := endToken
			if matchOnly {
				endName--
			}

			tokenName := pattern[beginName:endName]
			pe.segments = append(pe.segments, &TokenSegment{
				name:             tokenName,
				delimiter:        delimiter,
				matchOnly:        matchOnly,
				table:            pe.table,
				preserveRawValue: preserveRawValues,
			})
		}

		scanIndex = endToken + 1
	}
}

// Match attempts to match path against this expression.
// Reference: PatternExpression.Match
func (pe *PatternExpression) Match(path string, propertyDefinitions map[string]*PropertyDefinition) *ContentItem {
	var item *ContentItem
	startIndex := 0

	for _, segment := range pe.segments {
		endIndex, ok := segment.TryMatch(&item, path, propertyDefinitions, startIndex)
		if !ok {
			return nil
		}
		startIndex = endIndex
	}

	// Check if we consumed the entire path
	if startIndex != len(path) {
		return nil
	}

	// Apply defaults
	if item == nil {
		item = &ContentItem{
			Path:       path,
			Properties: pe.defaults,
		}
	} else {
		for key, value := range pe.defaults {
			item.Add(key, value)
		}
	}

	return item
}

// TryMatch for LiteralSegment
func (ls *LiteralSegment) TryMatch(item **ContentItem, path string, properties map[string]*PropertyDefinition, startIndex int) (int, bool) {
	if startIndex+len(ls.text) > len(path) {
		return 0, false
	}

	// Case-insensitive comparison
	pathSegment := path[startIndex : startIndex+len(ls.text)]
	if !strings.EqualFold(pathSegment, ls.text) {
		return 0, false
	}

	return startIndex + len(ls.text), true
}

// TryMatch for TokenSegment
func (ts *TokenSegment) TryMatch(item **ContentItem, path string, properties map[string]*PropertyDefinition, startIndex int) (int, bool) {
	// Find end of this token (until delimiter or end of path)
	endIndex := startIndex
	if ts.delimiter != 0 {
		for endIndex < len(path) && path[endIndex] != ts.delimiter {
			endIndex++
		}
	} else {
		endIndex = len(path)
	}

	if endIndex == startIndex && !ts.matchOnly {
		// Empty value for non-optional token
		return 0, false
	}

	// Get property definition
	propDef, ok := properties[ts.name]
	if !ok {
		// Unknown property, treat as string
		tokenValue := path[startIndex:endIndex]
		if *item == nil {
			*item = &ContentItem{
				Path:       path,
				Properties: make(map[string]interface{}),
			}
		}
		(*item).Properties[ts.name] = tokenValue
		return endIndex, true
	}

	// Try to parse value
	tokenValue := path[startIndex:endIndex]
	value, matched := propDef.TryLookup(tokenValue, ts.table, ts.matchOnly)

	if !matched {
		return 0, false
	}

	// Store value if not match-only
	if !ts.matchOnly && value != nil {
		if *item == nil {
			*item = &ContentItem{
				Path:       path,
				Properties: make(map[string]interface{}),
			}
		}

		// Store parsed value
		(*item).Properties[ts.name] = value

		// Store raw value if preserving
		if ts.preserveRawValue {
			(*item).Properties[ts.name+"_raw"] = tokenValue
		}
	}

	return endIndex, true
}
