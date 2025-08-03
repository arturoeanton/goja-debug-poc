package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
)

type DebugAdapter struct {
	reader   *bufio.Reader
	writer   io.Writer
	seq      int
	seqMutex sync.Mutex

	// Goja runtime
	runtime     *goja.Runtime
	debugger    *goja.Debugger
	program     string
	sourceCode  string
	sourceLines []string

	// Debug state
	running     bool
	terminated  bool
	breakpoints map[string][]int
	bpIDCounter int
	bpMap       map[int]*Breakpoint

	// Thread simulation (goja is single-threaded)
	threadID int

	// Variable references
	varRefCounter int
	varRefMap     map[int]interface{}

	// Synchronization
	debugStateMutex sync.Mutex
	waitingForCmd   bool
	nextCommand     goja.DebugCommand
	commandReady    chan struct{}

	// Debug state tracking
	currentState    *goja.DebuggerState
	currentLine     int  // Track current line (persistent across states)
	isPaused        bool // Track if we're paused
	
	// Logger
	logger *log.Logger
	logFile *os.File
}

func NewDebugAdapter(reader io.Reader, writer io.Writer) *DebugAdapter {
	// Initialize logger
	logFile, err := os.OpenFile("dap-adapter.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	var logger *log.Logger
	if err != nil {
		logger = log.New(io.Discard, "[DAP] ", log.Ldate|log.Ltime|log.Lmicroseconds)
	} else {
		logger = log.New(logFile, "[DAP] ", log.Ldate|log.Ltime|log.Lmicroseconds)
	}

	return &DebugAdapter{
		reader:        bufio.NewReader(reader),
		writer:        writer,
		seq:           1,
		breakpoints:   make(map[string][]int),
		bpMap:         make(map[int]*Breakpoint),
		varRefMap:     make(map[int]interface{}),
		threadID:      1,
		commandReady:  make(chan struct{}),
		nextCommand:   goja.DebugContinue,
		varRefCounter: 1000, // Start from 1000 to avoid conflicts
		currentLine:   -1,
		logger:        logger,
		logFile:       logFile,
	}
}

func (da *DebugAdapter) Close() {
	if da.logFile != nil {
		da.logger.Println("=== DAP Adapter Closed ===")
		da.logFile.Close()
	}
}

func (da *DebugAdapter) nextSeq() int {
	da.seqMutex.Lock()
	defer da.seqMutex.Unlock()
	seq := da.seq
	da.seq++
	return seq
}

func (da *DebugAdapter) sendResponse(requestSeq int, command string, success bool, body interface{}) {
	response := Response{
		Message: Message{
			Seq:  da.nextSeq(),
			Type: "response",
		},
		RequestSeq: requestSeq,
		Success:    success,
		Command:    command,
		Body:       body,
	}

	if !success && body == nil {
		response.ErrorMessage = "Unknown error"
	}

	da.logger.Printf("<<< Sending response: %s (success=%v)", command, success)
	da.sendMessage(response)
}

func (da *DebugAdapter) sendEvent(event string, body interface{}) {
	evt := Event{
		Message: Message{
			Seq:  da.nextSeq(),
			Type: "event",
		},
		Event: event,
		Body:  body,
	}

	da.logger.Printf("<<< EVENT: %s", event)
	da.sendMessage(evt)
}

func (da *DebugAdapter) sendMessage(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		da.logger.Printf("Error marshaling message: %v", err)
		return
	}

	// DAP uses Content-Length header
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	da.writer.Write([]byte(header))
	da.writer.Write(data)
}

func (da *DebugAdapter) readMessage() (*Request, error) {
	// Read Content-Length header
	headerLine, err := da.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(headerLine, "Content-Length:") {
		return nil, fmt.Errorf("invalid header: %s", headerLine)
	}

	lengthStr := strings.TrimSpace(strings.TrimPrefix(headerLine, "Content-Length:"))
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, err
	}

	// Read empty line
	da.reader.ReadString('\n')

	// Read message body
	body := make([]byte, length)
	_, err = io.ReadFull(da.reader, body)
	if err != nil {
		return nil, err
	}

	var req Request
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}

	return &req, nil
}

func (da *DebugAdapter) Run() {
	da.logger.Println("=== DAP Adapter Started ===")
	defer da.Close()

	for {
		req, err := da.readMessage()
		if err != nil {
			if err == io.EOF {
				break
			}
			da.logger.Printf("Error reading message: %v", err)
			continue
		}

		da.handleRequest(req)

		if da.terminated {
			break
		}
	}
}

func (da *DebugAdapter) handleRequest(req *Request) {
	da.logger.Printf(">>> Received request: %s", req.Command)

	switch req.Command {
	case "initialize":
		da.handleInitialize(req)
	case "launch":
		da.handleLaunch(req)
	case "setBreakpoints":
		da.handleSetBreakpoints(req)
	case "configurationDone":
		da.handleConfigurationDone(req)
	case "threads":
		da.handleThreads(req)
	case "stackTrace":
		da.handleStackTrace(req)
	case "scopes":
		da.handleScopes(req)
	case "variables":
		da.handleVariables(req)
	case "evaluate":
		da.handleEvaluate(req)
	case "continue":
		da.handleContinue(req)
	case "next":
		da.handleNext(req)
	case "stepIn":
		da.handleStepIn(req)
	case "stepOut":
		da.handleStepOut(req)
	case "pause":
		da.handlePause(req)
	case "disconnect":
		da.handleDisconnect(req)
	case "terminate":
		da.handleTerminate(req)
	default:
		da.logger.Printf("Unknown command: %s", req.Command)
		da.sendResponse(req.Seq, req.Command, false, nil)
	}
}

func (da *DebugAdapter) handleInitialize(req *Request) {
	capabilities := Capabilities{
		SupportsConfigurationDoneRequest: true,
		SupportsConditionalBreakpoints:   false,
		SupportsEvaluateForHovers:        true,
		SupportsSetVariable:              false,
		SupportsTerminateRequest:         true,
	}

	da.sendResponse(req.Seq, req.Command, true, capabilities)
	da.sendEvent("initialized", InitializedEvent{})
}

func (da *DebugAdapter) handleLaunch(req *Request) {
	var args LaunchRequestArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	da.program = args.Program
	da.logger.Printf("Program to debug: %s", da.program)

	// Read the program file
	content, err := os.ReadFile(da.program)
	if err != nil {
		da.sendResponse(req.Seq, req.Command, false, map[string]string{
			"error": fmt.Sprintf("Failed to read program: %v", err),
		})
		return
	}

	da.sourceCode = string(content)
	da.sourceLines = strings.Split(da.sourceCode, "\n")

	// Create runtime with debug mode enabled
	opts := goja.RuntimeOptions{
		EnableDebugMode: true,
	}
	da.runtime = goja.NewWithOptions(opts)
	da.debugger = da.runtime.EnableDebugger()

	// Debugger might not have SetLogger method in this version
	// da.debugger.SetLogger(da.logger)

	// Set up console.log
	console := da.runtime.NewObject()
	console.Set("log", func(args ...interface{}) {
		var output strings.Builder
		for i, arg := range args {
			if i > 0 {
				output.WriteString(" ")
			}
			output.WriteString(fmt.Sprintf("%v", arg))
		}
		da.sendEvent("output", map[string]interface{}{
			"category": "console",
			"output":   output.String() + "\n",
		})
	})
	da.runtime.Set("console", console)

	// Set up debug handler
	da.debugger.SetHandler(da.debugHandler)

	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handleSetBreakpoints(req *Request) {
	var args SetBreakpointsArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	filename := args.Source.Path
	if filename == "" {
		filename = args.Source.Name
	}

	// Normalize filename
	if filepath.Base(filename) == filepath.Base(da.program) {
		filename = da.program
	}

	da.logger.Printf("SetBreakpoints for file: %s, count: %d", filename, len(args.Breakpoints))

	// Clear existing breakpoints for this file
	// ClearBreakpoints might not exist, we'll track and remove manually
	if da.debugger != nil {
		// Remove each tracked breakpoint
		for id := range da.bpMap {
			da.debugger.RemoveBreakpoint(id)
		}
	}

	// Clear tracking maps
	da.bpMap = make(map[int]*Breakpoint)
	da.breakpoints[filename] = []int{}

	// Add new breakpoints
	var breakpoints []Breakpoint

	for _, sbp := range args.Breakpoints {
		if da.debugger != nil {
			gojaID := da.debugger.AddBreakpoint(filename, sbp.Line, sbp.Column)
			da.logger.Printf("Added breakpoint: line=%d, gojaID=%d", sbp.Line, gojaID)
		}

		da.bpIDCounter++
		bpID := da.bpIDCounter

		bp := Breakpoint{
			ID:       bpID,
			Verified: true,
			Source:   args.Source,
			Line:     sbp.Line,
			Column:   sbp.Column,
		}

		da.bpMap[bpID] = &bp
		da.breakpoints[filename] = append(da.breakpoints[filename], sbp.Line)
		breakpoints = append(breakpoints, bp)
	}

	da.sendResponse(req.Seq, req.Command, true, SetBreakpointsResponseBody{
		Breakpoints: breakpoints,
	})
}

func (da *DebugAdapter) handleConfigurationDone(req *Request) {
	da.sendResponse(req.Seq, req.Command, true, nil)

	// Parse the source to find the first executable line
	program, err := parser.ParseFile(nil, da.program, da.sourceCode, 0)
	if err != nil {
		da.logger.Printf("Parse error: %v", err)
		da.sendEvent("output", map[string]interface{}{
			"category": "stderr",
			"output":   fmt.Sprintf("Parse error: %v\n", err),
		})
		return
	}

	// Find first statement and set initial breakpoint
	for _, stmt := range program.Body {
		if pos := stmt.Idx0(); pos >= 0 {
			file := program.File
			filePos := file.Position(int(pos))
			id := da.debugger.AddBreakpoint(filePos.Filename, filePos.Line, filePos.Column)
			da.logger.Printf("Added initial breakpoint #%d at %s:%d:%d", id, filePos.Filename, filePos.Line, filePos.Column)
			break
		}
	}

	go da.runScript()
}

func (da *DebugAdapter) runScript() {
	da.running = true
	da.logger.Printf("Starting script execution...")

	// Compile with debug mode
	compiled, err := goja.Compile(da.program, da.sourceCode, true)
	if err != nil {
		da.logger.Printf("Compile error: %v", err)
		da.sendEvent("output", map[string]interface{}{
			"category": "stderr",
			"output":   fmt.Sprintf("Compile error: %v\n", err),
		})
		da.sendEvent("exited", ExitedEventBody{ExitCode: 1})
		da.sendEvent("terminated", TerminatedEventBody{})
		return
	}

	// Execute the program
	val, err := da.runtime.RunProgram(compiled)
	if err != nil {
		da.logger.Printf("Runtime error: %v", err)
		da.sendEvent("output", map[string]interface{}{
			"category": "stderr",
			"output":   fmt.Sprintf("Runtime error: %v\n", err),
		})
		da.sendEvent("exited", ExitedEventBody{ExitCode: 1})
		da.sendEvent("terminated", TerminatedEventBody{})
		return
	}

	da.logger.Printf("Script completed, result: %v", val)
	da.running = false
	da.sendEvent("exited", ExitedEventBody{ExitCode: 0})
	da.sendEvent("terminated", TerminatedEventBody{})
}

func (da *DebugAdapter) handleThreads(req *Request) {
	threads := []Thread{
		{
			ID:   da.threadID,
			Name: "main",
		},
	}

	da.sendResponse(req.Seq, req.Command, true, ThreadsResponseBody{
		Threads: threads,
	})
}

func (da *DebugAdapter) handleStackTrace(req *Request) {
	var args StackTraceArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	var frames []StackFrame

	// Use debug stack if available
	if da.currentState != nil && da.currentState.DebugStack != nil && len(da.currentState.DebugStack) > 0 {
		for i, frame := range da.currentState.DebugStack {
			funcName := frame.FuncName()
			if funcName == "" || funcName == "<native>" {
				if i == len(da.currentState.DebugStack)-1 {
					funcName = "<main>"
				} else {
					funcName = "<anonymous>"
				}
			}
			
			// Get position from frame - frame.Position() returns file.Position
			// Try to use SourcePos from the state first for the current frame
			line := 0
			column := 0
			filename := da.program
			
			if i == 0 && da.currentState != nil && da.currentState.SourcePos.Line > 0 {
				// Use current state position for top frame
				line = da.currentState.SourcePos.Line
				column = da.currentState.SourcePos.Column
				filename = da.currentState.SourcePos.Filename
			} else {
				// For other frames, try to parse the position
				// Note: This is a workaround as we don't have direct access to Position fields
				srcName := frame.SrcName()
				if srcName != "" && srcName != "<native>" {
					filename = srcName
				}
			}
			
			frames = append(frames, StackFrame{
				ID:     i + 1,
				Name:   funcName,
				Line:   line,
				Column: column,
				Source: Source{
					Name: filepath.Base(filename),
					Path: filename,
				},
			})
		}
	} else if da.currentState != nil && da.currentState.SourcePos.Line > 0 {
		// Fallback to current position only
		frames = append(frames, StackFrame{
			ID:     1,
			Name:   "main",
			Line:   da.currentState.SourcePos.Line,
			Column: da.currentState.SourcePos.Column,
			Source: Source{
				Name: filepath.Base(da.currentState.SourcePos.Filename),
				Path: da.currentState.SourcePos.Filename,
			},
		})
	} else if da.currentLine > 0 {
		// Use last known position
		frames = append(frames, StackFrame{
			ID:     1,
			Name:   "main",
			Line:   da.currentLine,
			Column: 0,
			Source: Source{
				Name: filepath.Base(da.program),
				Path: da.program,
			},
		})
	} else {
		// Default frame
		frames = append(frames, StackFrame{
			ID:     1,
			Name:   "main",
			Line:   1,
			Column: 0,
			Source: Source{
				Name: filepath.Base(da.program),
				Path: da.program,
			},
		})
	}

	da.sendResponse(req.Seq, req.Command, true, StackTraceResponseBody{
		StackFrames: frames,
		TotalFrames: len(frames),
	})
}

func (da *DebugAdapter) handleScopes(req *Request) {
	var args ScopesArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	var scopes []Scope

	// Always provide default scopes for now
	// TODO: Use actual scope information when available from debugger
	da.varRefCounter++
	localRef := da.varRefCounter
	da.varRefMap[localRef] = map[string]interface{}{
		"type":    "local",
		"frameID": args.FrameID,
	}

	scopes = append(scopes, Scope{
		Name:               "Local",
		VariablesReference: localRef,
		Expensive:          false,
	})

	da.varRefCounter++
	globalRef := da.varRefCounter
	da.varRefMap[globalRef] = map[string]interface{}{
		"type": "global",
	}

	scopes = append(scopes, Scope{
		Name:               "Global",
		VariablesReference: globalRef,
		Expensive:          false,
	})

	da.sendResponse(req.Seq, req.Command, true, ScopesResponseBody{
		Scopes: scopes,
	})
}

func (da *DebugAdapter) handleVariables(req *Request) {
	var args VariablesArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	da.logger.Printf("handleVariables: variablesReference=%d", args.VariablesReference)

	var variables []Variable

	if ref, ok := da.varRefMap[args.VariablesReference]; ok {
		switch v := ref.(type) {
		case int:
			// This would be a scope reference from the debugger
			// For now, just return empty since we don't have access to GetVariables
			// TODO: Implement when debugger API is available
		case map[string]interface{}:
			// Legacy scope info
			scopeType := v["type"].(string)
			if scopeType == "global" {
				variables = da.getGlobalVariables()
			}
		}
	}

	da.sendResponse(req.Seq, req.Command, true, VariablesResponseBody{
		Variables: variables,
	})
}

func (da *DebugAdapter) getGlobalVariables() []Variable {
	var variables []Variable

	if da.runtime != nil {
		globalObj := da.runtime.GlobalObject()
		if globalObj != nil {
			for _, key := range globalObj.Keys() {
				if da.isBuiltIn(key) {
					continue
				}

				val := globalObj.Get(key)
				if val != nil {
					value := da.formatValue(val)
					varType := da.getValueType(val)
					varRef := 0

					// Create reference for complex types
					if obj, ok := val.(*goja.Object); ok {
						if _, isFunc := goja.AssertFunction(obj); !isFunc {
							da.varRefCounter++
							varRef = da.varRefCounter
							da.varRefMap[varRef] = val
						}
					}

					variables = append(variables, Variable{
						Name:               key,
						Value:              value,
						Type:               varType,
						VariablesReference: varRef,
					})
				}
			}
		}
	}

	return variables
}

func (da *DebugAdapter) formatValue(val goja.Value) string {
	if val == nil || goja.IsUndefined(val) {
		return "undefined"
	}
	if goja.IsNull(val) {
		return "null"
	}

	// Check for special types
	if obj, ok := val.(*goja.Object); ok {
		className := obj.ClassName()
		switch className {
		case "Array":
			return "[Array]"
		case "Object":
			return "{Object}"
		case "Function":
			return "<Function>"
		default:
			return fmt.Sprintf("<%s>", className)
		}
	}

	// For primitives
	str := val.String()
	switch val.ExportType().Kind() {
	case reflect.String:
		return fmt.Sprintf(`"%s"`, str)
	default:
		return str
	}
}

func (da *DebugAdapter) getValueType(val goja.Value) string {
	if goja.IsUndefined(val) {
		return "undefined"
	}
	if goja.IsNull(val) {
		return "null"
	}

	switch val.Export().(type) {
	case string:
		return "string"
	case int, int64, float64:
		return "number"
	case bool:
		return "boolean"
	default:
		if obj, ok := val.(*goja.Object); ok {
			if _, ok := goja.AssertFunction(obj); ok {
				return "function"
			}
			if arr := obj.Export(); arr != nil && reflect.TypeOf(arr).Kind() == reflect.Slice {
				return "array"
			}
			return "object"
		}
		return "unknown"
	}
}

func (da *DebugAdapter) isBuiltIn(name string) bool {
	builtins := []string{
		"console", "Object", "Function", "Array", "String", "Number",
		"Boolean", "Date", "JSON", "Math", "RegExp", "Error",
		"undefined", "Infinity", "NaN", "parseInt", "parseFloat",
		"isNaN", "isFinite", "eval", "decodeURI", "decodeURIComponent",
		"encodeURI", "encodeURIComponent", "escape", "unescape",
	}

	for _, b := range builtins {
		if name == b {
			return true
		}
	}
	return false
}

func (da *DebugAdapter) handleEvaluate(req *Request) {
	var args EvaluateArguments
	if req.Arguments != nil {
		data, _ := json.Marshal(req.Arguments)
		json.Unmarshal(data, &args)
	}

	da.logger.Printf("Evaluating: '%s' in context: %s", args.Expression, args.Context)

	// Check if we're trying to evaluate while paused
	if da.isPaused && da.waitingForCmd {
		// Try to evaluate by accessing variables directly instead of running code
		if val := da.trySimpleEvaluation(args.Expression); val != nil {
			value := da.formatValue(val)
			valueType := da.getValueType(val)
			varRef := 0

			// Create reference for complex types
			if obj, ok := val.(*goja.Object); ok {
				if _, isFunc := goja.AssertFunction(obj); !isFunc {
					da.varRefCounter++
					varRef = da.varRefCounter
					da.varRefMap[varRef] = val
				}
			}

			da.sendResponse(req.Seq, req.Command, true, EvaluateResponseBody{
				Result:             value,
				Type:               valueType,
				VariablesReference: varRef,
			})
			return
		}
		
		// For complex expressions, return an error instead of deadlocking
		da.logger.Printf("Cannot evaluate complex expression while paused: %s", args.Expression)
		da.sendResponse(req.Seq, req.Command, false, map[string]string{
			"error": "Cannot evaluate complex expressions while paused. Try simple variable names only.",
		})
		return
	}

	// If not paused, evaluate normally
	result, err := da.runtime.RunString(args.Expression)
	if err != nil {
		da.logger.Printf("Evaluation error: %v", err)
		da.sendResponse(req.Seq, req.Command, false, map[string]string{
			"error": err.Error(),
		})
		return
	}

	value := da.formatValue(result)
	valueType := da.getValueType(result)
	varRef := 0

	// Create reference for complex types
	if obj, ok := result.(*goja.Object); ok {
		if _, isFunc := goja.AssertFunction(obj); !isFunc {
			da.varRefCounter++
			varRef = da.varRefCounter
			da.varRefMap[varRef] = result
		}
	}

	da.sendResponse(req.Seq, req.Command, true, EvaluateResponseBody{
		Result:             value,
		Type:               valueType,
		VariablesReference: varRef,
	})
}

// trySimpleEvaluation attempts to evaluate simple variable names without running code
func (da *DebugAdapter) trySimpleEvaluation(expr string) goja.Value {
	// Trim whitespace
	expr = strings.TrimSpace(expr)
	
	// Only handle simple identifiers for now
	if !isSimpleIdentifier(expr) {
		return nil
	}
	
	// Try to get from global object
	globalObj := da.runtime.GlobalObject()
	if globalObj != nil {
		if val := globalObj.Get(expr); val != nil && !goja.IsUndefined(val) {
			return val
		}
	}
	
	return nil
}

// isSimpleIdentifier checks if a string is a simple JavaScript identifier
func isSimpleIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	
	// Check first character
	if !((s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z') || s[0] == '_' || s[0] == '$') {
		return false
	}
	
	// Check remaining characters
	for i := 1; i < len(s); i++ {
		if !((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '_' || s[i] == '$') {
			return false
		}
	}
	
	return true
}

func (da *DebugAdapter) handleContinue(req *Request) {
	da.logger.Printf("handleContinue: Setting command to Continue")
	
	da.debugStateMutex.Lock()
	da.nextCommand = goja.DebugContinue
	da.isPaused = false
	
	if da.waitingForCmd {
		da.waitingForCmd = false
		close(da.commandReady)
		da.commandReady = make(chan struct{})
	}
	da.debugStateMutex.Unlock()

	da.sendResponse(req.Seq, req.Command, true, ContinueResponseBody{
		AllThreadsContinued: true,
	})
}

func (da *DebugAdapter) handleNext(req *Request) {
	da.logger.Printf("handleNext: Setting command to StepOver")
	
	da.debugStateMutex.Lock()
	da.nextCommand = goja.DebugStepOver
	da.isPaused = false
	
	if da.waitingForCmd {
		da.waitingForCmd = false
		close(da.commandReady)
		da.commandReady = make(chan struct{})
	}
	da.debugStateMutex.Unlock()

	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handleStepIn(req *Request) {
	da.logger.Printf("handleStepIn: Setting command to StepInto")
	
	da.debugStateMutex.Lock()
	da.nextCommand = goja.DebugStepInto
	da.isPaused = false
	
	if da.waitingForCmd {
		da.waitingForCmd = false
		close(da.commandReady)
		da.commandReady = make(chan struct{})
	}
	da.debugStateMutex.Unlock()

	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handleStepOut(req *Request) {
	da.logger.Printf("handleStepOut: Setting command to StepOut")
	
	da.debugStateMutex.Lock()
	da.nextCommand = goja.DebugStepOut
	da.isPaused = false
	
	if da.waitingForCmd {
		da.waitingForCmd = false
		close(da.commandReady)
		da.commandReady = make(chan struct{})
	}
	da.debugStateMutex.Unlock()

	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handlePause(req *Request) {
	da.logger.Printf("handlePause: Setting command to Pause")
	da.debugger.Pause()
	da.sendResponse(req.Seq, req.Command, true, nil)
}

func (da *DebugAdapter) handleDisconnect(req *Request) {
	da.sendResponse(req.Seq, req.Command, true, nil)
	da.terminated = true
}

func (da *DebugAdapter) handleTerminate(req *Request) {
	da.sendResponse(req.Seq, req.Command, true, nil)
	da.sendEvent("terminated", TerminatedEventBody{})
	da.terminated = true
}

func (da *DebugAdapter) debugHandler(state *goja.DebuggerState) goja.DebugCommand {
	da.logger.Printf("DebugHandler: Called - PC=%d, Line=%d, File=%s, StepMode=%v, InNative=%v, NativeName=%s",
		state.PC, state.SourcePos.Line, state.SourcePos.Filename, state.StepMode, state.InNativeCall, state.NativeFunctionName)
	
	// Log stack information for debugging
	if state.DebugStack != nil && len(state.DebugStack) > 0 {
		da.logger.Printf("DebugHandler: Stack depth=%d, top frame: %s", len(state.DebugStack), state.DebugStack[0].FuncName)
	}
	
	if state.Breakpoint != nil {
		da.logger.Printf("DebugHandler: Hit breakpoint ID=%d at line %d", state.Breakpoint.ID(), state.Breakpoint.SourcePos.Line)
	}
	
	// Store current state
	da.debugStateMutex.Lock()
	da.currentState = state
	
	// Only update currentLine if we have a valid line
	if state.SourcePos.Line > 0 {
		da.currentLine = state.SourcePos.Line
	}
	
	da.isPaused = true
	da.debugStateMutex.Unlock()

	// Handle internal code during stepping
	if state.StepMode && state.SourcePos.Line == 0 && !state.InNativeCall {
		// We're in internal code, continue with the same step mode
		da.logger.Printf("DebugHandler: In internal code, continuing with step mode")
		time.Sleep(50 * time.Millisecond) // Brief pause like the console example
		return goja.DebugStepInto
	}
	
	// Skip stopping in native calls when not stepping
	if !state.StepMode && state.InNativeCall {
		da.logger.Printf("DebugHandler: In native call, continuing")
		return goja.DebugContinue
	}

	// Send stopped event
	reason := "step"
	if state.Breakpoint != nil {
		reason = "breakpoint"
	}
	
	da.sendEvent("stopped", StoppedEventBody{
		Reason:            reason,
		ThreadID:          da.threadID,
		AllThreadsStopped: true,
	})

	// Wait for command
	da.debugStateMutex.Lock()
	da.waitingForCmd = true
	da.debugStateMutex.Unlock()

	da.logger.Printf("DebugHandler: Waiting for command...")
	<-da.commandReady

	da.debugStateMutex.Lock()
	cmd := da.nextCommand
	da.debugStateMutex.Unlock()

	da.logger.Printf("DebugHandler: Returning command: %v", cmd)
	return cmd
}