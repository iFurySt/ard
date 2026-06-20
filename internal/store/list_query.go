package store

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func ParseListFilterExpression(expression string) (ListFilter, error) {
	var filter ListFilter
	if strings.TrimSpace(expression) == "" {
		return filter, nil
	}
	tokens, err := tokenizeListFilterExpression(expression)
	if err != nil {
		return ListFilter{}, err
	}
	parser := listFilterParser{tokens: tokens, filter: &filter}
	parsed, err := parser.parseExpression()
	if err != nil {
		return ListFilter{}, err
	}
	if parser.hasMore() {
		return ListFilter{}, fmt.Errorf("unexpected filter token %q", parser.peek().Value)
	}
	filter.Expression = &parsed
	return filter, nil
}

type listFilterToken struct {
	Kind  string
	Value string
}

type listFilterParser struct {
	tokens []listFilterToken
	index  int
	filter *ListFilter
}

func (parser *listFilterParser) parseExpression() (ListFilterExpression, error) {
	return parser.parseOrExpression()
}

func (parser *listFilterParser) parseOrExpression() (ListFilterExpression, error) {
	left, err := parser.parseAndExpression()
	if err != nil {
		return ListFilterExpression{}, err
	}
	children := []ListFilterExpression{left}
	for parser.matchWord("OR") {
		right, err := parser.parseAndExpression()
		if err != nil {
			return ListFilterExpression{}, err
		}
		children = append(children, right)
	}
	if len(children) == 1 {
		return left, nil
	}
	return ListFilterExpression{Operator: "OR", Children: children}, nil
}

func (parser *listFilterParser) parseAndExpression() (ListFilterExpression, error) {
	left, err := parser.parsePrimaryExpression()
	if err != nil {
		return ListFilterExpression{}, err
	}
	children := []ListFilterExpression{left}
	for parser.matchWord("AND") {
		right, err := parser.parsePrimaryExpression()
		if err != nil {
			return ListFilterExpression{}, err
		}
		children = append(children, right)
	}
	if len(children) == 1 {
		return left, nil
	}
	return ListFilterExpression{Operator: "AND", Children: children}, nil
}

func (parser *listFilterParser) parsePrimaryExpression() (ListFilterExpression, error) {
	if parser.matchKind("(") {
		inner, err := parser.parseExpression()
		if err != nil {
			return ListFilterExpression{}, err
		}
		if !parser.matchKind(")") {
			return ListFilterExpression{}, errors.New("filter expression has an unmatched parenthesis")
		}
		return inner, nil
	}
	clause, err := parser.parseClause()
	if err != nil {
		return ListFilterExpression{}, err
	}
	return ListFilterExpression{Operator: "CLAUSE", Clause: &clause}, nil
}

func (parser *listFilterParser) parseClause() (ListFilterClause, error) {
	field, err := parser.expectValue("filter field")
	if err != nil {
		return ListFilterClause{}, err
	}
	operator, err := parser.expectOperator()
	if err != nil {
		return ListFilterClause{}, err
	}
	values, err := parser.parseValues()
	if err != nil {
		return ListFilterClause{}, err
	}
	clause, err := parser.listFilterClause(field, operator, values)
	if err != nil {
		return ListFilterClause{}, err
	}
	parser.filter.Clauses = append(parser.filter.Clauses, clause)
	return clause, nil
}

func (parser *listFilterParser) parseValues() ([]string, error) {
	values := []string{}
	for {
		value, err := parser.expectValue("filter value")
		if err != nil {
			return nil, err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, errors.New("filter values must not be empty")
		}
		values = append(values, value)
		if !parser.matchKind(",") {
			break
		}
	}
	return values, nil
}

func (parser *listFilterParser) listFilterClause(field string, operator string, values []string) (ListFilterClause, error) {
	switch field {
	case "displayName":
		if !listFilterOperatorAllowed(operator, "=", "!=", "contains") {
			return ListFilterClause{}, unsupportedListFilterOperator(field, "=", "!=", "contains")
		}
		clause := listFilterClause(field, "", operator, values)
		if operator == "=" {
			parser.filter.DisplayName = append(parser.filter.DisplayName, values...)
		}
		return clause, nil
	case "type":
		if !listFilterOperatorAllowed(operator, "=", "!=", "contains") {
			return ListFilterClause{}, unsupportedListFilterOperator(field, "=", "!=", "contains")
		}
		clause := listFilterClause(field, "", operator, values)
		if operator == "=" {
			parser.filter.Types = append(parser.filter.Types, values...)
		}
		return clause, nil
	case "publisherId":
		if !listFilterOperatorAllowed(operator, "=", "!=", "contains") {
			return ListFilterClause{}, unsupportedListFilterOperator(field, "=", "!=", "contains")
		}
		clause := listFilterClause(field, "", operator, values)
		if operator == "=" {
			parser.filter.PublisherIDs = append(parser.filter.PublisherIDs, values...)
		}
		return clause, nil
	case "tags":
		if !listFilterOperatorAllowed(operator, "=", "!=", "contains") {
			return ListFilterClause{}, unsupportedListFilterOperator(field, "=", "!=", "contains")
		}
		clause := listFilterClause(field, "", operator, values)
		if operator == "=" {
			parser.filter.Tags = append(parser.filter.Tags, values...)
		}
		return clause, nil
	case "capabilities":
		if !listFilterOperatorAllowed(operator, "=", "!=", "contains") {
			return ListFilterClause{}, unsupportedListFilterOperator(field, "=", "!=", "contains")
		}
		clause := listFilterClause(field, "", operator, values)
		if operator == "=" {
			parser.filter.Capabilities = append(parser.filter.Capabilities, values...)
		}
		return clause, nil
	case "createdAfter":
		return parser.listFilterTimeClause(field, operator, values)
	case "updatedAfter":
		return parser.listFilterTimeClause(field, operator, values)
	default:
		if strings.HasPrefix(field, "metadata.") {
			return parser.listFilterMetadataClause(field, operator, values)
		}
		return ListFilterClause{}, fmt.Errorf("unsupported filter field %q", field)
	}
}

func (parser *listFilterParser) listFilterTimeClause(field string, operator string, values []string) (ListFilterClause, error) {
	if !listFilterOperatorAllowed(operator, ">", ">=") {
		return ListFilterClause{}, unsupportedListFilterOperator(field, ">", ">=")
	}
	timestamp, err := singleListFilterTime(field, values)
	if err != nil {
		return ListFilterClause{}, err
	}
	clause := listFilterTimeClause(field, operator, timestamp)
	if operator == ">" {
		if field == "createdAfter" {
			parser.filter.CreatedAfter = &timestamp
		} else {
			parser.filter.UpdatedAfter = &timestamp
		}
	}
	return clause, nil
}

func (parser *listFilterParser) listFilterMetadataClause(field string, operator string, values []string) (ListFilterClause, error) {
	if !listFilterOperatorAllowed(operator, "=", "!=", "contains") {
		return ListFilterClause{}, unsupportedListFilterOperator(field, "=", "!=", "contains")
	}
	key, err := listMetadataKey(field)
	if err != nil {
		return ListFilterClause{}, err
	}
	clause := listFilterClause("metadata", key, operator, values)
	if parser.filter.Metadata == nil {
		parser.filter.Metadata = map[string][]string{}
	}
	if operator == "=" {
		parser.filter.Metadata[key] = append(parser.filter.Metadata[key], values...)
	}
	return clause, nil
}

func (parser *listFilterParser) expectValue(label string) (string, error) {
	if !parser.hasMore() {
		return "", fmt.Errorf("expected %s", label)
	}
	token := parser.next()
	if token.Kind != "value" {
		return "", fmt.Errorf("expected %s, got %q", label, token.Value)
	}
	return token.Value, nil
}

func (parser *listFilterParser) expectOperator() (string, error) {
	if !parser.hasMore() {
		return "", errors.New("expected filter operator")
	}
	token := parser.next()
	if token.Kind == "operator" {
		return token.Value, nil
	}
	if token.Kind == "value" && strings.EqualFold(token.Value, "contains") {
		return "contains", nil
	}
	return "", fmt.Errorf("expected filter operator, got %q", token.Value)
}

func (parser *listFilterParser) matchKind(kind string) bool {
	if !parser.hasMore() || parser.peek().Kind != kind {
		return false
	}
	parser.index++
	return true
}

func (parser *listFilterParser) matchWord(word string) bool {
	if !parser.hasMore() {
		return false
	}
	token := parser.peek()
	if token.Kind != "value" || !strings.EqualFold(token.Value, word) {
		return false
	}
	parser.index++
	return true
}

func (parser *listFilterParser) hasMore() bool {
	return parser.index < len(parser.tokens)
}

func (parser *listFilterParser) peek() listFilterToken {
	return parser.tokens[parser.index]
}

func (parser *listFilterParser) next() listFilterToken {
	token := parser.tokens[parser.index]
	parser.index++
	return token
}

func tokenizeListFilterExpression(expression string) ([]listFilterToken, error) {
	tokens := []listFilterToken{}
	for index := 0; index < len(expression); {
		char := expression[index]
		if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
			index++
			continue
		}
		switch char {
		case '(', ')', ',':
			tokens = append(tokens, listFilterToken{Kind: string(char), Value: string(char)})
			index++
		case '\'', '"':
			value, next, err := readQuotedListFilterToken(expression, index)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, listFilterToken{Kind: "value", Value: value})
			index = next
		case '=', '!', '>':
			operator, next, err := readListFilterOperator(expression, index)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, listFilterToken{Kind: "operator", Value: operator})
			index = next
		default:
			value, next := readBareListFilterToken(expression, index)
			if value == "" {
				return nil, fmt.Errorf("unexpected filter character %q", char)
			}
			tokens = append(tokens, listFilterToken{Kind: "value", Value: value})
			index = next
		}
	}
	if len(tokens) == 0 {
		return nil, errors.New("filter must not be empty")
	}
	return tokens, nil
}

func readQuotedListFilterToken(expression string, start int) (string, int, error) {
	quote := expression[start]
	var builder strings.Builder
	for index := start + 1; index < len(expression); index++ {
		if expression[index] == quote {
			return builder.String(), index + 1, nil
		}
		builder.WriteByte(expression[index])
	}
	return "", 0, errors.New("filter expression has an unterminated quoted value")
}

func readListFilterOperator(expression string, start int) (string, int, error) {
	if strings.HasPrefix(expression[start:], ">=") || strings.HasPrefix(expression[start:], "!=") {
		return expression[start : start+2], start + 2, nil
	}
	if expression[start] == '>' || expression[start] == '=' {
		return expression[start : start+1], start + 1, nil
	}
	return "", 0, fmt.Errorf("unsupported filter operator starting at %q", expression[start:])
}

func readBareListFilterToken(expression string, start int) (string, int) {
	index := start
	for index < len(expression) {
		char := expression[index]
		if char == ' ' || char == '\t' || char == '\n' || char == '\r' || char == '(' || char == ')' || char == ',' || char == '=' || char == '!' || char == '>' {
			break
		}
		index++
	}
	return expression[start:index], index
}

func ParseListOrderBy(raw string) (ListOrder, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ListOrder{}, nil
	}
	parts := strings.Fields(raw)
	if len(parts) < 1 || len(parts) > 2 {
		return ListOrder{}, errors.New("orderBy must be a field optionally followed by ASC or DESC")
	}
	field, err := normalizeListOrderField(parts[0])
	if err != nil {
		return ListOrder{}, err
	}
	direction := "ASC"
	if len(parts) == 2 {
		direction = strings.ToUpper(parts[1])
		if direction != "ASC" && direction != "DESC" {
			return ListOrder{}, errors.New("orderBy direction must be ASC or DESC")
		}
	}
	return ListOrder{Field: field, Direction: direction}, nil
}

func singleListFilterTime(field string, values []string) (time.Time, error) {
	if len(values) != 1 {
		return time.Time{}, fmt.Errorf("filter field %q requires exactly one timestamp", field)
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, values[0]); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("filter field %q requires an ISO 8601 timestamp", field)
}

func listMetadataKey(field string) (string, error) {
	key := strings.TrimPrefix(field, "metadata.")
	if key == "" {
		return "", errors.New("metadata filter key must not be empty")
	}
	for _, char := range key {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		switch char {
		case '_', '-', '.':
			continue
		default:
			return "", fmt.Errorf("metadata filter key %q contains unsupported character %q", key, char)
		}
	}
	return key, nil
}

func listFilterClause(field string, metadataKey string, operator string, values []string) ListFilterClause {
	return ListFilterClause{
		Field:       field,
		MetadataKey: metadataKey,
		Operator:    operator,
		Values:      append([]string(nil), values...),
	}
}

func listFilterTimeClause(field string, operator string, timestamp time.Time) ListFilterClause {
	return ListFilterClause{
		Field:    field,
		Operator: operator,
		Time:     &timestamp,
	}
}

func listFilterOperatorAllowed(operator string, allowed ...string) bool {
	for _, candidate := range allowed {
		if operator == candidate {
			return true
		}
	}
	return false
}

func unsupportedListFilterOperator(field string, allowed ...string) error {
	return fmt.Errorf("filter field %q only supports %s", field, strings.Join(allowed, ", "))
}

func normalizeListOrderField(field string) (string, error) {
	switch field {
	case "displayName", "display_name", "name":
		return "displayName", nil
	case "type":
		return "type", nil
	case "createdAt", "created_at":
		return "createdAt", nil
	case "updatedAt", "updated_at":
		return "updatedAt", nil
	case "publisherId", "publisher_id":
		return "publisherId", nil
	default:
		return "", fmt.Errorf("unsupported orderBy field %q", field)
	}
}
