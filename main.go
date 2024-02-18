package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const JSON_QUOTE = '"'
const JSON_LEFT_BRACKET = '['
const JSON_RIGHT_BRACKET = ']'
const JSON_LEFT_BRACE = '{'
const JSON_RIGHT_BRACE = '}'
const JSON_COMMA = ','
const JSON_COLON = ':'
const JSON_NULL = "null"
const JSON_TRUE = "true"
const JSON_FALSE = "false"
const JSON_PERIOD = '.'
const JSON_MINUS = '-'

var JSON_WHITESPACE = [...]rune{ ' ', '\t', '\b', '\n', '\r' }

var JSON_TOKENS = [...]rune{ 
	JSON_LEFT_BRACKET,
	JSON_RIGHT_BRACKET,
	JSON_LEFT_BRACE,
	JSON_RIGHT_BRACE,
	JSON_COMMA,
	JSON_COLON,
}

type Token struct {
	value string
	tokenType string
}

func main() {
	file, err := os.Open("example.json")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer file.Close()

	buffer := make([]byte, 4096)
	bytesRead := 0
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			log.Fatal(err.Error())
		}
		bytesRead += n
		if n == 0 {
			break
		}
	}
	
	jsonString := string(buffer[:bytesRead])
	tokens, err := lexerAnalysis(jsonString)
	if err != nil {
		log.Fatal(err.Error())
	}
	// fmt.Println(tokens)
	// os.Exit(1)
	data, _, err := parse(tokens)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(data)
}

func parse(tokens []Token) (any, int, error) {
		token := tokens[0]
		if token.value == string(JSON_LEFT_BRACKET) {
			return parseArray(tokens)
		}
		if token.value == string(JSON_LEFT_BRACE) {
			return parseObject(tokens)
		}
		if token.value == string(JSON_COLON) || token.value == string(JSON_COMMA) {
			return nil, 0, errors.New("Invalid token!")
		}
		switch token.value {
		case JSON_NULL:
			return nil, 1, nil
		case JSON_TRUE:
			return true, 1, nil
		case JSON_FALSE:
			return false, 1, nil
		default:
			if token.tokenType == "NUMBER" {
				if strings.Contains(token.value, ".") {
					tokenFloat, err := strconv.ParseFloat(token.value, 64)
					if err != nil {
						panic(err)
					}
					return tokenFloat, 1, nil
				}
				tokenInt, err := strconv.Atoi(token.value)
				if err != nil {
					panic(err)
				}
				return tokenInt, 1, nil
			}
			return token.value, 1, nil
		}
}

func parseArray(tokens []Token) (any, int, error) {
	jsonArray := []any{}
	for i := 1; i < len(tokens); i++ {
		token := tokens[i]
		if token.value == string(JSON_RIGHT_BRACKET) {
			return jsonArray, i+1, nil
		}
		if token.value == string(JSON_COMMA) {
			continue
		}
		data, l, err := parse(tokens[i:])
		if err != nil {
			return nil, i+l, err
		}
		i += l - 1
		jsonArray = append(jsonArray, data)
	}
	return nil, 0, errors.New("Expected end of array!")
}

func parseObject(tokens []Token) (any, int, error) {
	jsonObject := make(map[string]any)
	for i := 1; i < len(tokens); i++ {
		token := tokens[i]
		if token.value == string(JSON_RIGHT_BRACE) {
			return jsonObject, i+1, nil
		}
		if token.value == string(JSON_COLON) {
			prevToken := tokens[i-1]
			// nextToken := tokens[i+1]
			if prevToken.tokenType != "STRING" {
				return nil, i+1, errors.New("Invalid syntax: prev not str")
			}
			key := tokens[i-1]
			i++
			val, l, err := parse(tokens[i:])
			// fmt.Println("l: " + strconv.Itoa(l))
			if err != nil {
				return nil, i+l, err
			}
			jsonObject[key.value] = val
			i += l - 1
			continue
		}
		if token.value == string(JSON_COMMA) {
			continue
		}
		if token.tokenType == "STRING" {
			continue
		}
		return nil, i, errors.New("Unexpected token: " + token.value)
	}
	return nil, 0, nil
}

func lexerAnalysis(jsonString string) ([]Token, error) {
	jsonBytes := []byte(jsonString)

	var tokens []Token

	outer:
	for i := 0; i < len(jsonBytes); i++ {
		c := jsonBytes[i]

		if c == byte(JSON_QUOTE) {
			i++
			var token strings.Builder
			for i < len(jsonBytes) {
				b := jsonBytes[i]
				if b == JSON_QUOTE {
					
					tokens = append(tokens, Token{token.String(), "STRING"})
					continue outer
				}
				token.WriteByte(b)
				i++
			}
			return nil, errors.New("Expected end of string!")
		}
	
		if c == byte(JSON_MINUS) || unicode.IsDigit(rune(c))  {
			var token strings.Builder
			if c == byte('0') {
				return nil, errors.New("Unexpected token!.")
			}
			token.WriteByte(c)
			i++
			for ;i < len(jsonBytes); i++ {
				b := jsonBytes[i]
				if unicode.IsDigit(rune(b)) {
					token.WriteByte(b)
				} else if b == byte(JSON_PERIOD) {
					prev := rune(jsonBytes[i-1])
					if i+1 == len(jsonBytes) {
						return nil, errors.New("Unexpected token!.")
					}
					next := rune(jsonBytes[i+1])
					if !unicode.IsDigit(prev) || !unicode.IsDigit(next) {
						return nil, errors.New("Unexpected token!.")
					}
					token.WriteByte(b)
				} else {
					tokens = append(tokens, Token{token.String(), "NUMBER"})
					i--
					continue outer
				}
			}
		}

		for _, token := range JSON_TOKENS {
			if c == byte(token) {
				tokens = append(tokens, Token{string(c), "SYMBOL"})
				continue outer
			}
		}

		for _, ws := range JSON_WHITESPACE {
			if c == byte(ws) {
				continue outer
			}
		}

		remainingBytesLen := len(jsonBytes[i:])
		if remainingBytesLen >= len(JSON_NULL) {
			token := string(jsonBytes[i:i+len(JSON_NULL)])
			if token == JSON_NULL {
				tokens = append(tokens, Token{token, "KEYWORD"})
				i = i + len(JSON_NULL) - 1
				continue
			}
		}
		if remainingBytesLen >= len(JSON_TRUE) {
			token := string(jsonBytes[i:i+len(JSON_TRUE)])
			if token == JSON_TRUE {
				tokens = append(tokens, Token{token, "KEYWORD"})
				i = i + len(JSON_TRUE) - 1
				continue
			}
		}
		if remainingBytesLen >= len(JSON_FALSE) {
			token := string(jsonBytes[i:i+len(JSON_FALSE)])
			if token == JSON_FALSE {
				tokens = append(tokens, Token{token, "KEYWORD"})
				i = i + len(JSON_FALSE) - 1
				continue
			}
		} 
		return nil, errors.New("Unexpected token!")
	}
	return tokens, nil
}
