package mux

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

type ExactMux struct {
	mutex           sync.RWMutex
	routes          map[string]map[string]exactMuxEntry
	paramRoutes     map[string][]paramsMuxEntry
	notFoundHandler http.Handler
}

type paramsMuxEntry struct {
	http.HandlerFunc
	handler   http.Handler
	pathParts []string
}

type Middleware func(handler http.HandlerFunc) http.HandlerFunc

func NewExactMux() *ExactMux {
	return &ExactMux{}
}

func (m *ExactMux) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if handler, err := m.handler(request.Method, request.URL.Path); err == nil {
		handler.ServeHTTP(writer, request)
	}

	if m.notFoundHandler != nil {
		m.notFoundHandler.ServeHTTP(writer, request)
	}
}

func (m *ExactMux) GET(
	pattern string,
	handlerFunc http.HandlerFunc,
	middlewares ...Middleware,
) {
	m.HandleFuncWithMiddlewares(
		http.MethodGet,
		pattern,
		handlerFunc,
		middlewares...,
	)
}

func (m *ExactMux) POST(
	pattern string,
	handlerFunc http.HandlerFunc,
	middlewares ...Middleware,
) {
	m.HandleFuncWithMiddlewares(
		http.MethodPost,
		pattern,
		handlerFunc,
		middlewares...,
	)
}

func (m *ExactMux) PUT(
	pattern string,
	handlerFunc http.HandlerFunc,
	middlewares ...Middleware,
) {
	m.HandleFuncWithMiddlewares(
		http.MethodPut,
		pattern,
		handlerFunc,
		middlewares...,
	)
}

func (m *ExactMux) DELETE(
	pattern string,
	handlerFunc http.HandlerFunc,
	middlewares ...Middleware,
) {
	m.HandleFuncWithMiddlewares(
		http.MethodDelete,
		pattern,
		handlerFunc,
		middlewares...,
	)
}

func (m *ExactMux) HandleFuncWithMiddlewares(
	method string,
	pattern string,
	handlerFunc http.HandlerFunc,
	middlewares ...Middleware,
) {
	for _, middleware := range middlewares {
		handlerFunc = middleware(handlerFunc)
	}
	m.HandleFunc(method, pattern, handlerFunc)
}

func (m *ExactMux) handler(method string, path string) (handler http.Handler, err error) {
	entries, exists := m.routes[method]
	if exists {
		if entry, ok := entries[path]; ok {
			return entry.handler, nil
		}
	}
	paramEntries, exists := m.paramRoutes[method]
	if !exists {
		return nil, fmt.Errorf("can't find handler for: %s, %s", method, path)
	}

	for _, entry := range paramEntries {
		if paramRoutesMatch(entry, path) {
			return entry.handler, nil
		}
	}
	return nil, fmt.Errorf("can't find handler for: %s, %s", method, path)
}
func (m *ExactMux) HandleFunc(method string, pattern string, handlerFunc func(responseWriter http.ResponseWriter, request *http.Request)) {
	if !strings.HasPrefix(pattern, "/") {
		panic(fmt.Errorf("pattern must start with /: %s", pattern))
	}

	if handlerFunc == nil {
		panic(errors.New("handler can't be empty"))
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !isPathWithParams(pattern) {
		entry := exactMuxEntry{
			pattern: pattern,
			handler: http.HandlerFunc(handlerFunc),
			weight:  calculateWeight(pattern),
		}

		if _, exists := m.routes[method][pattern]; exists {
			panic(fmt.Errorf("ambigious mapping: %s", pattern))
		}

		if m.routes == nil {
			m.routes = make(map[string]map[string]exactMuxEntry)
		}

		if m.routes[method] == nil {
			m.routes[method] = make(map[string]exactMuxEntry)
		}

		m.routes[method][pattern] = entry
		return
	}

	pathParts := strings.Split(pattern, "/")
	entry := paramsMuxEntry{
		handler:   m.ParamFunc(handlerFunc, pathParts),
		pathParts: pathParts,
	}

	if _, exists := m.routes[method][pattern]; exists {
		panic(fmt.Errorf("ambigious mapping: %s", pattern))
	}

	if m.paramRoutes == nil {
		m.paramRoutes = make(map[string][]paramsMuxEntry)
	}

	if m.paramRoutes[method] == nil {
		m.paramRoutes[method] = make([]paramsMuxEntry, 0)
	}

	m.paramRoutes[method] = append(m.paramRoutes[method], entry)
	return
}

func getParameterNames(parts []string) []string {
	params := make([]string, 0)

	for _, part := range parts {
		if len(part) > 0 {
			if part[firstSymbol] == '{' {
				params = append(params, part[1:len(part)-1])
			}
		}
	}

	return params
}

func isPathWithParams(pattern string) bool {
	if strings.Contains(pattern, "{") {
		if strings.Contains(pattern, "}") {
			return true
		}
	}
	return false
}

func (m *ExactMux) ParamFunc(handler func(w http.ResponseWriter, r *http.Request), pathParts []string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		request = addAllParametersToRequest(request.URL.Path, request, pathParts)
		handler(writer, request)
	}
}

func addAllParametersToRequest(path string, request *http.Request, pathParts []string) *http.Request {
	parametersValues := getParametersValues(pathParts, path)
	parametersNames := getParameterNames(pathParts)

	if len(parametersValues) != len(parametersNames) {
		log.Fatal("paramsValues and paramsNames must have same item count : ", len(parametersValues), len(pathParts))
	}

	for index := range parametersNames {
		request = request.WithContext(context.WithValue(request.Context(),
			parametersNames[index], parametersValues[index]))
	}

	return request
}

const firstSymbol = 0

func paramRoutesMatch(entry paramsMuxEntry, path string) (ok bool) {
	pathSplit := strings.Split(path, "/")
	if len(pathSplit) != len(entry.pathParts) {
		return false
	}
	for index, _ := range entry.pathParts {
		if len(entry.pathParts[index]) > 0 {
			if entry.pathParts[index][firstSymbol] != '{' {
				if entry.pathParts[index] != pathSplit[index] {
					return false
				}
			}
		}
	}
	return true
}
func getParametersValues(templateParts []string, path string) (values []string) {
	pathSplit := strings.Split(path, "/")
	if len(pathSplit) != len(templateParts) {
		return nil
	}
	for index, _ := range templateParts {
		if len(templateParts[index]) > 0 {
			if templateParts[index][firstSymbol] == '{' {
				values = append(values, pathSplit[index])
			}
		}
	}
	return values
}

type exactMuxEntry struct {
	pattern string
	handler http.Handler
	weight  int
}

func calculateWeight(pattern string) int {
	if pattern == "/" {
		return 0
	}

	count := (strings.Count(pattern, "/") - 1) * 2
	if !strings.HasSuffix(pattern, "/") {
		return count + 1
	}
	return count
}
