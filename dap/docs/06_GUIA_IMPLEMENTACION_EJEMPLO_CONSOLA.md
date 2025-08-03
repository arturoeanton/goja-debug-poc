# Guía de Implementación del Debugger de Goja - Consola Interactiva

## Estructura del Proyecto

Crear la siguiente estructura en `examples/debugger/goja-debug/`:

```
examples/
└── debugger/
    └── goja-debug/
        ├── main.go              # Consola principal
        ├── go.mod               # Módulo Go
        ├── run-debugger.sh      # Script para ejecutar
        ├── ejemplo_debug.js     # Script de ejemplo
        └── *.md                 # Documentación
```

## Archivo: go.mod

```go
module github.com/dop251/goja/examples/debugger/goja-debug

go 1.16

require (
    github.com/dop251/goja v0.0.0-00010101000000-000000000000
    github.com/fatih/color v1.13.0
    golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
)

replace github.com/dop251/goja => ../../../
```

## Archivo: main.go (Completo)

```go
package main

import (
    "bufio"
    "flag"
    "fmt"
    "io"
    "log"
    "os"
    "reflect"
    "runtime"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/dop251/goja"
    "github.com/dop251/goja/parser"
    "github.com/fatih/color"
    "golang.org/x/term"
)

// DebugConsole es la consola principal del debugger
// Maneja la interfaz de usuario y la interacción con el debugger de Goja
type DebugConsole struct {
    // Runtime y debugger de Goja
    runtime        *goja.Runtime
    debugger       *goja.Debugger
    
    // Información del archivo fuente
    currentFile    string      // Archivo JS actual
    source         string      // Código fuente completo
    sourceLines    []string    // Líneas del código fuente
    
    // Estado de ejecución
    isRunning      bool        // Si el programa está ejecutándose
    currentLine    int         // Línea actual de ejecución
    mu             sync.Mutex  // Mutex para acceso concurrente
    isPaused       bool        // Si está pausado en un breakpoint
    currentState   *goja.DebuggerState // Estado actual del debugger
    
    // Buffer de consola para console.log
    consoleBuffer  []string    // Mensajes de console.log
    consoleMaxSize int         // Tamaño máximo del buffer
    
    // Buffer separado para mensajes del debugger
    debugBuffer    []string    // Mensajes del debugger (info, errores, etc)
    lastCommand    string      // Último comando ejecutado para repetir con Enter
    
    // Logger para debugging
    logger         *log.Logger // Logger para debug del debugger
    logFile        *os.File    // Archivo de log
    
    // Input/Output
    reader         *bufio.Reader // Para leer comandos
    commandHistory []string      // Historial de comandos
    historyIndex   int          // Índice actual en el historial
    
    // Estado de la UI
    termWidth      int         // Ancho del terminal
    termHeight     int         // Alto del terminal
    localScroll    int         // Scroll en ventana de variables locales
    globalScroll   int         // Scroll en ventana de variables globales
    codeScroll     int         // Scroll en ventana de código
    showStack      bool        // Mostrar stack trace
    activePane     int         // 0=code, 1=locals, 2=globals, 3=console
    showGlobals    bool        // Mostrar variables globales
    varFilter      string      // Filtro para variables
}

// Constantes para los estilos de cajas
const (
    boxSingle = iota
    boxDouble
)

var (
    // Caracteres para dibujar cajas
    boxChars = map[int]map[string]string{
        boxSingle: {
            "horizontal": "─", "vertical": "│",
            "topLeft": "┌", "topRight": "┐",
            "bottomLeft": "└", "bottomRight": "┘",
            "cross": "┼", "teeDown": "┬", "teeUp": "┴",
            "teeRight": "├", "teeLeft": "┤",
        },
        boxDouble: {
            "horizontal": "═", "vertical": "║",
            "topLeft": "╔", "topRight": "╗",
            "bottomLeft": "╚", "bottomRight": "╝",
            "cross": "╬", "teeDown": "╦", "teeUp": "╩",
            "teeRight": "╠", "teeLeft": "╣",
        },
    }
)

// NewDebugConsole crea una nueva instancia de la consola de debugging
func NewDebugConsole() *DebugConsole {
    dc := &DebugConsole{
        consoleMaxSize: 100,
        reader:         bufio.NewReader(os.Stdin),
        showStack:      false,
        activePane:     0,
        showGlobals:    false,  // Por defecto muestra locales
        commandHistory: make([]string, 0),
        historyIndex:   -1,
        varFilter:      "",     // Sin filtro por defecto
    }

    // Inicializar logger
    logFile, err := os.OpenFile("goja.debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Warning: Could not create log file: %v\n", err)
        dc.logger = log.New(io.Discard, "[GOJA_DEBUG] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
    } else {
        dc.logFile = logFile
        dc.logger = log.New(logFile, "[GOJA_DEBUG] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
        dc.logger.Println("=== Goja Debug Console Started ===")
        dc.logger.Printf("Version: %s\n", runtime.Version())
    }

    // Obtener tamaño del terminal
    dc.updateTerminalSize()

    return dc
}

// updateTerminalSize actualiza las dimensiones del terminal
func (dc *DebugConsole) updateTerminalSize() {
    width, height, err := term.GetSize(int(os.Stdout.Fd()))
    if err != nil {
        dc.termWidth = 80
        dc.termHeight = 25
    } else {
        dc.termWidth = width
        dc.termHeight = height
    }
}

// Close cierra recursos abiertos
func (dc *DebugConsole) Close() {
    if dc.logFile != nil {
        dc.logger.Println("=== Goja Debug Console Closed ===")
        dc.logFile.Close()
    }
}

// LoadFile carga un archivo JavaScript para debuggear
func (dc *DebugConsole) LoadFile(filename string) error {
    dc.logger.Printf("LoadFile: Loading file %s\n", filename)
    dc.currentFile = filename
    content, err := os.ReadFile(filename)
    if err != nil {
        dc.logger.Printf("LoadFile: Error reading file: %v\n", err)
        return err
    }
    dc.source = string(content)
    dc.sourceLines = strings.Split(dc.source, "\n")
    dc.logger.Printf("LoadFile: Successfully loaded %d lines from %s\n", len(dc.sourceLines), filename)
    return nil
}

// Run ejecuta el programa JavaScript con el debugger habilitado
func (dc *DebugConsole) Run() error {
    // Inicializar runtime con debugger
    dc.logger.Println("Run: Initializing runtime with debug mode enabled")
    opts := goja.RuntimeOptions{
        EnableDebugMode: true,
    }
    dc.runtime = goja.NewWithOptions(opts)
    dc.debugger = dc.runtime.EnableDebugger()
    dc.logger.Printf("Run: Runtime and debugger initialized successfully")

    // Configurar console.log para capturar la salida
    console := dc.runtime.NewObject()
    console.Set("log", func(args ...interface{}) {
        dc.mu.Lock()
        defer dc.mu.Unlock()
        parts := make([]string, len(args))
        for i, arg := range args {
            parts[i] = fmt.Sprint(arg)
        }
        msg := strings.Join(parts, " ")
        dc.logger.Printf("console.log: %s\n", msg)
        dc.consoleBuffer = append(dc.consoleBuffer, msg)
        if len(dc.consoleBuffer) > dc.consoleMaxSize {
            dc.consoleBuffer = dc.consoleBuffer[1:]
        }
    })
    dc.runtime.Set("console", console)

    // Configurar el manejador de debugging
    dc.debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
        dc.logger.Printf("DebugHandler: Called - PC=%d, Line=%d, File=%s, StepMode=%v, InNative=%v, NativeName=%s\n",
            state.PC, state.SourcePos.Line, state.SourcePos.Filename, state.StepMode, state.InNativeCall, state.NativeFunctionName)
        
        if state.Breakpoint != nil {
            dc.logger.Printf("DebugHandler: Hit breakpoint ID=%d at line %d\n", state.Breakpoint.ID(), state.Breakpoint.SourcePos.Line)
        }
        
        dc.mu.Lock()
        dc.isRunning = false
        // Solo actualizar currentLine si tenemos una línea válida
        if state.SourcePos.Line > 0 {
            dc.currentLine = state.SourcePos.Line
        }
        dc.isPaused = true
        dc.currentState = state
        dc.mu.Unlock()

        // Si estamos en código interno durante un step, mostrar brevemente y continuar
        if state.StepMode && state.SourcePos.Line == 0 && !state.InNativeCall {
            // Mostrar estado brevemente
            dc.displayState(state)
            
            // Mostrar mensaje en la consola
            dc.consoleBuffer = append(dc.consoleBuffer, 
                fmt.Sprintf("[Pasando por código interno PC:%d]", state.PC))
            if len(dc.consoleBuffer) > dc.consoleMaxSize {
                dc.consoleBuffer = dc.consoleBuffer[1:]
            }
            
            // Esperar un momento muy breve para que se vea
            time.Sleep(50 * time.Millisecond)
            
            // Continuar con el mismo modo de step
            return goja.DebugStepInto
        }

        // Mostrar estado actual
        dc.displayState(state)

        // Leer comando del usuario
        cmd := dc.readCommand()
        dc.logger.Printf("DebugHandler: Returning command: %v\n", cmd)
        return cmd
    })

    // Parsear el código fuente
    dc.logger.Printf("Run: Parsing source file %s\n", dc.currentFile)
    program, err := parser.ParseFile(nil, dc.currentFile, dc.source, 0)
    if err != nil {
        dc.logger.Printf("Run: Parse error: %v\n", err)
        return fmt.Errorf("parse error: %v", err)
    }
    dc.logger.Println("Run: Source parsed successfully")

    // Compilar el programa
    dc.logger.Printf("Run: Compiling program with debug mode\n")
    compiled, err := goja.Compile(dc.currentFile, dc.source, true)
    if err != nil {
        dc.logger.Printf("Run: Compile error: %v\n", err)
        return fmt.Errorf("compile error: %v", err)
    }
    dc.logger.Println("Run: Program compiled successfully")

    // Encontrar puntos de entrada principales y establecer breakpoint inicial
    for _, stmt := range program.Body {
        if pos := stmt.Idx0(); pos >= 0 {
            file := program.File
            filePos := file.Position(int(pos))
            id := dc.debugger.AddBreakpoint(filePos.Filename, filePos.Line, filePos.Column)
            dc.logger.Printf("Run: Added initial breakpoint #%d at %s:%d:%d\n", id, filePos.Filename, filePos.Line, filePos.Column)
            fmt.Printf("Added initial breakpoint #%d at line %d\n", id, filePos.Line)
            break
        }
    }

    // Limpiar pantalla y mostrar UI inicial
    dc.clearScreen()
    dc.drawInitialUI()

    // Ejecutar el programa en una goroutine separada
    dc.mu.Lock()
    dc.isRunning = true
    dc.mu.Unlock()

    dc.logger.Println("Run: Starting program execution in goroutine")
    go func() {
        dc.logger.Println("Run: Program execution started")
        _, err := dc.runtime.RunProgram(compiled)
        if err != nil {
            dc.logger.Printf("Run: Program error: %v\n", err)
            dc.showError(fmt.Sprintf("Program error: %v", err))
        }
        dc.mu.Lock()
        dc.isRunning = false
        dc.mu.Unlock()
        dc.logger.Println("Run: Program execution finished")
        dc.moveCursor(dc.termHeight-1, 1)
        fmt.Println("\nProgram finished - Press Enter to exit")
        dc.reader.ReadString('\n')
        os.Exit(0)
    }()

    // Mantener el thread principal vivo
    select {}
}

// Métodos de UI...

// clearScreen limpia la pantalla
func (dc *DebugConsole) clearScreen() {
    fmt.Print("\033[2J\033[H")
}

// moveCursor mueve el cursor a una posición específica
func (dc *DebugConsole) moveCursor(row, col int) {
    fmt.Printf("\033[%d;%dH", row, col)
}

// setColor establece colores de texto y fondo
func (dc *DebugConsole) setColor(fg, bg color.Attribute) {
    fmt.Printf("\033[%d;%dm", bg+10, fg)
}

// resetColor resetea los colores al default
func (dc *DebugConsole) resetColor() {
    fmt.Print("\033[0m")
}

// drawBox dibuja una caja con caracteres especiales
func (dc *DebugConsole) drawBox(x, y, width, height int, title string, style int) {
    chars := boxChars[style]
    
    // Línea superior
    dc.moveCursor(y, x)
    fmt.Print(chars["topLeft"])
    if title != "" {
        titleStr := fmt.Sprintf(" %s ", title)
        titleLen := len(titleStr)
        padding := (width - 2 - titleLen) / 2
        for i := 0; i < padding; i++ {
            fmt.Print(chars["horizontal"])
        }
        dc.setColor(color.FgYellow, color.BgBlue)
        fmt.Print(titleStr)
        dc.resetColor()
        for i := padding + titleLen; i < width-2; i++ {
            fmt.Print(chars["horizontal"])
        }
    } else {
        for i := 0; i < width-2; i++ {
            fmt.Print(chars["horizontal"])
        }
    }
    fmt.Print(chars["topRight"])
    
    // Lados
    for i := 1; i < height-1; i++ {
        dc.moveCursor(y+i, x)
        fmt.Print(chars["vertical"])
        dc.moveCursor(y+i, x+width-1)
        fmt.Print(chars["vertical"])
    }
    
    // Línea inferior
    dc.moveCursor(y+height-1, x)
    fmt.Print(chars["bottomLeft"])
    for i := 0; i < width-2; i++ {
        fmt.Print(chars["horizontal"])
    }
    fmt.Print(chars["bottomRight"])
}

// drawInitialUI dibuja la UI inicial
func (dc *DebugConsole) drawInitialUI() {
    dc.updateTerminalSize()
    
    // Barra de título
    dc.moveCursor(1, 1)
    dc.setColor(color.FgBlack, color.BgCyan)
    fmt.Printf(" GOJA DEBUG - %s%s", dc.currentFile, strings.Repeat(" ", dc.termWidth-len(dc.currentFile)-12))
    dc.resetColor()
}

// displayState muestra el estado actual del debugger en la UI
func (dc *DebugConsole) displayState(state *goja.DebuggerState) {
    dc.logger.Printf("displayState: Called for line %d, PC=%d\n", state.SourcePos.Line, state.PC)
    
    dc.clearScreen()
    dc.updateTerminalSize()
    
    // Calcular layout
    halfWidth := dc.termWidth / 2
    codeHeight := dc.termHeight - 14       // Altura completa para código
    consoleHeight := 6
    
    // Determinar la línea actual a mostrar
    // Si la línea es 0 o estamos en código nativo, usar la última línea conocida
    displayLine := state.SourcePos.Line
    if displayLine == 0 && dc.currentLine > 0 {
        displayLine = dc.currentLine
    }
    
    // Actualizar currentLine solo si tenemos una línea válida
    if state.SourcePos.Line > 0 {
        dc.currentLine = state.SourcePos.Line
    }
    
    // Barra de título con información del estado
    dc.moveCursor(1, 1)
    dc.setColor(color.FgBlack, color.BgCyan)
    
    var title string
    if state.InNativeCall {
        title = fmt.Sprintf(" GOJA DEBUG - %s [Función Nativa: %s] PC:%d ", 
            dc.currentFile, state.NativeFunctionName, state.PC)
    } else if state.SourcePos.Line == 0 {
        title = fmt.Sprintf(" GOJA DEBUG - %s [Código interno] PC:%d ", 
            dc.currentFile, state.PC)
    } else {
        title = fmt.Sprintf(" GOJA DEBUG - %s Línea:%d PC:%d ", 
            dc.currentFile, state.SourcePos.Line, state.PC)
    }
    
    fmt.Print(title)
    fmt.Print(strings.Repeat(" ", dc.termWidth-len(title)))
    dc.resetColor()
    
    // Dibujar ventana de código (lado izquierdo)
    // Siempre mostrar el código en la última línea conocida
    dc.drawBox(1, 2, halfWidth, codeHeight, "Código Fuente", boxDouble)
    dc.displayCode(2, 3, halfWidth-2, codeHeight-2, displayLine)
    
    // Dibujar ventana de variables (derecha)
    // El título cambia según qué variables estamos mostrando
    varTitle := "Variables Locales [L]"
    if dc.showGlobals {
        varTitle = "Variables Globales [G]"
    }
    if dc.varFilter != "" {
        varTitle += fmt.Sprintf(" (filtro: %s)", dc.varFilter)
    }
    
    // Una sola ventana grande para variables
    varWindowHeight := dc.termHeight - 14  // Altura total para variables
    dc.drawBox(halfWidth+1, 2, halfWidth-1, varWindowHeight, varTitle, boxSingle)
    
    if dc.showGlobals {
        dc.displayGlobals(halfWidth+2, 3, halfWidth-3, varWindowHeight-2, state)
    } else {
        dc.displayLocals(halfWidth+2, 3, halfWidth-3, varWindowHeight-2, state)
    }
    
    // Dibujar ventana de consola (parte inferior)
    consoleY := dc.termHeight - consoleHeight - 6
    dc.drawBox(1, consoleY, dc.termWidth, consoleHeight, "Salida de Consola", boxSingle)
    dc.displayConsole(2, consoleY+1, dc.termWidth-2, consoleHeight-2)
    
    // Dibujar línea de comandos
    commandY := dc.termHeight - 5
    dc.drawBox(1, commandY, dc.termWidth, 5, "Comandos", boxSingle)
    dc.displayCommands(2, commandY+1)
}

// displayCode muestra el código fuente con la línea actual resaltada
func (dc *DebugConsole) displayCode(x, y, width, height int, currentLine int) {
    // Si no hay línea actual válida, mostrar mensaje informativo
    if currentLine == 0 {
        dc.moveCursor(y+height/2-1, x+5)
        dc.setColor(color.FgYellow, color.BgBlack)
        fmt.Print("[ Ejecutando código interno del motor ]")
        dc.resetColor()
        dc.moveCursor(y+height/2, x+5)
        fmt.Print("El debugger volverá al código fuente")
        dc.moveCursor(y+height/2+1, x+5)
        fmt.Print("cuando termine la operación actual.")
        return
    }
    
    // Calcular rango visible con el scroll aplicado
    start := currentLine - height/2 + dc.codeScroll
    if start < 1 {
        start = 1
        dc.codeScroll = 0 // Resetear scroll si está fuera de rango
    }
    end := start + height - 1
    if end > len(dc.sourceLines) {
        end = len(dc.sourceLines)
        start = end - height + 1
        if start < 1 {
            start = 1
        }
    }
    
    row := 0
    for i := start; i <= end && row < height; i++ {
        if i > len(dc.sourceLines) {
            break
        }
        
        dc.moveCursor(y+row, x)
        
        // Número de línea con indicador
        if i == currentLine {
            dc.setColor(color.FgYellow, color.BgBlue)
            fmt.Printf("→%4d ", i)
        } else {
            dc.setColor(color.FgCyan, color.BgBlack)
            fmt.Printf(" %4d ", i)
        }
        dc.resetColor()
        
        // Código
        line := dc.sourceLines[i-1]
        maxLen := width - 7
        if len(line) > maxLen {
            line = line[:maxLen-3] + "..."
        }
        
        // Resaltar línea actual
        if i == currentLine {
            dc.setColor(color.FgWhite, color.BgBlue)
        }
        fmt.Print(line)
        if i == currentLine && len(line) < maxLen {
            // Rellenar el resto de la línea para que el fondo se vea completo
            fmt.Print(strings.Repeat(" ", maxLen-len(line)))
        }
        dc.resetColor()
        
        row++
    }
    
    // Mostrar indicadores de scroll si hay más líneas
    if start > 1 || end < len(dc.sourceLines) {
        dc.moveCursor(y-1, x+width-15)
        dc.setColor(color.FgCyan, color.BgBlack)
        fmt.Printf("[↑↓ %d-%d/%d]", start, end, len(dc.sourceLines))
        dc.resetColor()
    }
}

// displayLocals muestra las variables locales del contexto actual
func (dc *DebugConsole) displayLocals(x, y, width, height int, state *goja.DebuggerState) {
    if len(state.DebugStack) == 0 {
        dc.moveCursor(y, x)
        fmt.Print("No local variables")
        return
    }
    
    var localVars []goja.Variable
    frame := state.DebugStack[0]
    for _, scope := range frame.Scopes {
        if scope.Name == "Local" || scope.Name == "Closure" {
            vars := dc.debugger.GetVariables(scope.VariablesRef)
            localVars = append(localVars, vars...)
        }
    }
    
    // Mostrar toggle de stack trace
    dc.moveCursor(y, x)
    dc.setColor(color.FgMagenta, color.BgBlack)
    fmt.Printf("[st=Stack] ")
    dc.resetColor()
    
    if dc.showStack {
        dc.displayStackTrace(x, y+1, width, height-1, state)
    } else {
        dc.displayVariableList(x, y+1, width, height-1, localVars, dc.localScroll)
    }
}

// displayGlobals muestra las variables globales
func (dc *DebugConsole) displayGlobals(x, y, width, height int, state *goja.DebuggerState) {
    if len(state.DebugStack) == 0 {
        dc.moveCursor(y, x)
        fmt.Print("No global variables")
        return
    }
    
    var globalVars []goja.Variable
    frame := state.DebugStack[0]
    for _, scope := range frame.Scopes {
        if scope.Name == "Global" {
            vars := dc.debugger.GetVariables(scope.VariablesRef)
            globalVars = append(globalVars, vars...)
        }
    }
    
    dc.displayVariableList(x, y, width, height, globalVars, dc.globalScroll)
}

// displayVariableList muestra una lista de variables con soporte para filtrado
func (dc *DebugConsole) displayVariableList(x, y, width, height int, vars []goja.Variable, scroll int) {
    // Filtrar variables si hay un filtro activo
    filteredVars := vars
    if dc.varFilter != "" {
        filteredVars = []goja.Variable{}
        filterLower := strings.ToLower(dc.varFilter)
        for _, v := range vars {
            if strings.Contains(strings.ToLower(v.Name), filterLower) {
                filteredVars = append(filteredVars, v)
            }
        }
    }
    
    if len(filteredVars) == 0 {
        dc.moveCursor(y, x)
        if dc.varFilter != "" {
            fmt.Print("(sin coincidencias)")
        } else {
            fmt.Print("(vacío)")
        }
        return
    }
    
    // Mostrar indicadores de scroll
    if len(filteredVars) > height {
        dc.moveCursor(y-1, x+width-10)
        dc.setColor(color.FgCyan, color.BgBlack)
        fmt.Printf("[%d/%d]", min(scroll+height, len(filteredVars)), len(filteredVars))
        dc.resetColor()
    }
    
    for i := 0; i < height && scroll+i < len(filteredVars); i++ {
        v := filteredVars[scroll+i]
        dc.moveCursor(y+i, x)
        
        // Nombre de variable
        dc.setColor(color.FgGreen, color.BgBlack)
        name := v.Name
        if len(name) > 15 {
            name = name[:12] + "..."
        }
        fmt.Printf("%-15s ", name)
        dc.resetColor()
        
        // Valor
        value := dc.formatValue(v.Value)
        maxValLen := width - 17
        if len(value) > maxValLen {
            value = value[:maxValLen-3] + "..."
        }
        fmt.Print(value)
    }
}

// displayStackTrace muestra el stack trace
func (dc *DebugConsole) displayStackTrace(x, y, width, height int, state *goja.DebuggerState) {
    dc.moveCursor(y, x)
    dc.setColor(color.FgYellow, color.BgBlack)
    fmt.Println("Call Stack:")
    dc.resetColor()
    
    for i := 0; i < height-1 && i < len(state.CallStack); i++ {
        frame := state.CallStack[i]
        dc.moveCursor(y+i+1, x)
        
        if i == 0 {
            dc.setColor(color.FgGreen, color.BgBlack)
            fmt.Print("→ ")
        } else {
            fmt.Print("  ")
        }
        
        funcName := frame.FuncName()
        if funcName == "" {
            funcName = "<anonymous>"
        }
        pos := frame.Position()
        
        line := fmt.Sprintf("%s at %s", funcName, pos)
        if len(line) > width-2 {
            line = line[:width-5] + "..."
        }
        fmt.Print(line)
        dc.resetColor()
    }
}

// displayConsole muestra la salida de console.log y mensajes de debug en dos columnas
func (dc *DebugConsole) displayConsole(x, y, width, height int) {
    halfWidth := (width - 3) / 2  // -3 para el separador " │ "
    
    // Headers
    dc.moveCursor(y-1, x)
    dc.setColor(color.FgYellow, color.BgBlack)
    fmt.Printf("%-*s │ %-*s", halfWidth, "Salida del Programa", halfWidth, "Mensajes del Debugger")
    dc.resetColor()
    
    // Mostrar salida del programa (columna izquierda)
    consoleStart := 0
    if len(dc.consoleBuffer) > height {
        consoleStart = len(dc.consoleBuffer) - height
    }
    
    // Mostrar mensajes del debugger (columna derecha)
    debugStart := 0
    if len(dc.debugBuffer) > height {
        debugStart = len(dc.debugBuffer) - height
    }
    
    for i := 0; i < height; i++ {
        dc.moveCursor(y+i, x)
        
        // Columna izquierda: salida del programa
        leftLine := ""
        if consoleStart+i < len(dc.consoleBuffer) {
            leftLine = dc.consoleBuffer[consoleStart+i]
            if len(leftLine) > halfWidth {
                leftLine = leftLine[:halfWidth-3] + "..."
            }
        }
        fmt.Printf("%-*s", halfWidth, leftLine)
        
        // Separador
        dc.setColor(color.FgCyan, color.BgBlack)
        fmt.Print(" │ ")
        dc.resetColor()
        
        // Columna derecha: mensajes del debugger
        rightLine := ""
        if debugStart+i < len(dc.debugBuffer) {
            rightLine = dc.debugBuffer[debugStart+i]
            if len(rightLine) > halfWidth {
                rightLine = rightLine[:halfWidth-3] + "..."
            }
        }
        fmt.Print(rightLine)
    }
}

// displayCommands muestra los comandos disponibles en la ventana de comandos
func (dc *DebugConsole) displayCommands(x, y int) {
    commands := []string{
        "F10=Siguiente  F11=Entrar  st=Stack  F8=Continuar  L=Locales  G=Globales",
        "c=continuar  n=siguiente  s=entrar  o=salir  b <línea>=break  p <expr>=evaluar",
        "f <texto>=filtrar variables  F=quitar filtro  ↑↓=historial  Enter=repetir  q=salir",
    }
    
    for i, cmd := range commands {
        dc.moveCursor(y+i, x)
        dc.setColor(color.FgCyan, color.BgBlack)
        fmt.Print(cmd)
        dc.resetColor()
    }
}

// formatValue formatea un valor para mostrar
func (dc *DebugConsole) formatValue(val goja.Value) string {
    if val == nil {
        return "undefined"
    }
    
    // Verificar tipos especiales
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
    
    // Para valores primitivos
    str := val.String()
    switch val.ExportType().Kind() {
    case reflect.String:
        return fmt.Sprintf(`"%s"`, str)
    default:
        return str
    }
}

// showError muestra un mensaje de error
func (dc *DebugConsole) showError(msg string) {
    y := dc.termHeight / 2
    width := len(msg) + 4
    x := (dc.termWidth - width) / 2
    
    dc.drawBox(x, y-1, width, 3, "Error", boxDouble)
    dc.moveCursor(y, x+2)
    dc.setColor(color.FgRed, color.BgBlack)
    fmt.Print(msg)
    dc.resetColor()
}

// readCommand lee un comando del usuario con soporte para historial
func (dc *DebugConsole) readCommand() goja.DebugCommand {
    // Posicionar cursor en el área de comandos
    dc.moveCursor(dc.termHeight-2, 3)
    fmt.Print("debug> ")
    
    // Buffer para construir el comando
    var cmdBuffer []rune
    cursorPos := 0
    
    // Configurar terminal en modo raw para capturar teclas especiales
    oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
    if err != nil {
        dc.logger.Printf("readCommand: Error setting raw mode: %v\n", err)
    } else {
        defer term.Restore(int(os.Stdin.Fd()), oldState)
    }
    
    for {
        // Leer un byte
        var buf [1]byte
        n, err := os.Stdin.Read(buf[:])
        if err != nil || n == 0 {
            continue
        }
        
        b := buf[0]
        
        // Manejar secuencias de escape (teclas especiales)
        if b == 27 { // ESC
            // Leer los siguientes bytes para identificar la tecla
            var seq [2]byte
            os.Stdin.Read(seq[:])
            
            if seq[0] == '[' {
                switch seq[1] {
                case 'A': // Flecha arriba - historial anterior
                    if dc.historyIndex > 0 {
                        dc.historyIndex--
                        cmdBuffer = []rune(dc.commandHistory[dc.historyIndex])
                        cursorPos = len(cmdBuffer)
                        // Limpiar línea y mostrar comando del historial
                        dc.moveCursor(dc.termHeight-2, 10)
                        fmt.Print(strings.Repeat(" ", 70))
                        dc.moveCursor(dc.termHeight-2, 10)
                        fmt.Print(string(cmdBuffer))
                    }
                case 'B': // Flecha abajo - historial siguiente
                    if dc.historyIndex < len(dc.commandHistory)-1 {
                        dc.historyIndex++
                        cmdBuffer = []rune(dc.commandHistory[dc.historyIndex])
                        cursorPos = len(cmdBuffer)
                        // Limpiar línea y mostrar comando del historial
                        dc.moveCursor(dc.termHeight-2, 10)
                        fmt.Print(strings.Repeat(" ", 70))
                        dc.moveCursor(dc.termHeight-2, 10)
                        fmt.Print(string(cmdBuffer))
                    } else if dc.historyIndex == len(dc.commandHistory)-1 {
                        // Volver a línea vacía
                        dc.historyIndex = len(dc.commandHistory)
                        cmdBuffer = []rune{}
                        cursorPos = 0
                        dc.moveCursor(dc.termHeight-2, 10)
                        fmt.Print(strings.Repeat(" ", 70))
                        dc.moveCursor(dc.termHeight-2, 10)
                    }
                }
            }
            continue
        }
        
        // Enter (CR o LF)
        if b == '\r' || b == '\n' {
            cmd := strings.TrimSpace(string(cmdBuffer))
            if cmd != "" {
                // Agregar al historial
                dc.commandHistory = append(dc.commandHistory, cmd)
                dc.historyIndex = len(dc.commandHistory)
            }
            dc.logger.Printf("readCommand: Received command: %s\n", cmd)
            
            // Restaurar terminal antes de procesar
            if oldState != nil {
                term.Restore(int(os.Stdin.Fd()), oldState)
            }
            
            // Procesar el comando
            return dc.processCommand(cmd)
        }
        
        // Backspace
        if b == 127 || b == 8 {
            if cursorPos > 0 {
                cmdBuffer = append(cmdBuffer[:cursorPos-1], cmdBuffer[cursorPos:]...)
                cursorPos--
                // Actualizar display
                dc.moveCursor(dc.termHeight-2, 10)
                fmt.Print(string(cmdBuffer) + " ")
                dc.moveCursor(dc.termHeight-2, 10+cursorPos)
            }
            continue
        }
        
        // Caracteres normales
        if b >= 32 && b < 127 {
            cmdBuffer = append(cmdBuffer[:cursorPos], append([]rune{rune(b)}, cmdBuffer[cursorPos:]...)...)
            cursorPos++
            // Actualizar display
            dc.moveCursor(dc.termHeight-2, 10)
            fmt.Print(string(cmdBuffer))
            dc.moveCursor(dc.termHeight-2, 10+cursorPos)
        }
    }
}

// processCommand procesa un comando y devuelve la acción correspondiente
func (dc *DebugConsole) processCommand(cmd string) goja.DebugCommand {
    if cmd == "" {
        // Si no hay comando, repetir el último
        if dc.lastCommand != "" {
            cmd = dc.lastCommand
            dc.debugBuffer = append(dc.debugBuffer, fmt.Sprintf("Repitiendo: %s", cmd))
        } else {
            return dc.readCommand()
        }
    } else {
        // Guardar el comando actual como último comando
        dc.lastCommand = cmd
    }
    
    parts := strings.Fields(cmd)
    if len(parts) == 0 {
        return dc.readCommand()
    }

    switch parts[0] {
    case "continue", "c", "continuar", "F8":
        dc.logger.Println("processCommand: Ejecutando continuar")
        return goja.DebugContinue
    case "step", "s", "stepinto", "si", "entrar", "F11":
        dc.logger.Println("processCommand: Ejecutando step into")
        return goja.DebugStepInto
    case "next", "n", "stepover", "so", "siguiente", "F10":
        dc.logger.Println("processCommand: Ejecutando step over")
        return goja.DebugStepOver
    case "out", "o", "stepout":
        dc.logger.Println("processCommand: Ejecutando step out")
        return goja.DebugStepOut
    case "break", "b", "F9":
        if len(parts) < 2 {
            dc.showError("Uso: break <línea>")
            return dc.readCommand()
        }
        line, err := strconv.Atoi(parts[1])
        if err != nil {
            dc.showError(fmt.Sprintf("Número de línea inválido: %s", parts[1]))
            return dc.readCommand()
        }
        id := dc.debugger.AddBreakpoint(dc.currentFile, line, 0)
        dc.logger.Printf("processCommand: Agregado breakpoint #%d en línea %d\n", id, line)
        // Mostrar mensaje en el buffer de debug
        dc.debugBuffer = append(dc.debugBuffer, fmt.Sprintf("Breakpoint #%d agregado en línea %d", id, line))
        dc.displayState(dc.currentState)
        return dc.readCommand()
    case "print", "p", "evaluar":
        if len(parts) < 2 {
            dc.showError("Uso: print <expresión>")
            return dc.readCommand()
        }
        expr := strings.Join(parts[1:], " ")
        dc.logger.Printf("processCommand: Evaluando expresión: %s\n", expr)
        dc.evaluateExpression(expr)
        return dc.readCommand()
    case "st", "stack", "trace":
        dc.showStack = !dc.showStack
        dc.displayState(dc.currentState)
        return dc.readCommand()
    
    // Comandos para cambiar entre variables locales y globales
    case "l", "L", "locales":
        dc.showGlobals = false
        dc.debugBuffer = append(dc.debugBuffer, "Mostrando variables locales")
        dc.displayState(dc.currentState)
        return dc.readCommand()
    
    case "g", "G", "globales":
        dc.showGlobals = true
        dc.debugBuffer = append(dc.debugBuffer, "Mostrando variables globales")
        dc.displayState(dc.currentState)
        return dc.readCommand()
    
    // Comandos para filtrar variables
    case "f", "filter", "filtrar":
        if len(parts) < 2 {
            dc.showError("Uso: f <texto> - filtra variables que contengan el texto")
            return dc.readCommand()
        }
        dc.varFilter = strings.Join(parts[1:], " ")
        dc.debugBuffer = append(dc.debugBuffer, fmt.Sprintf("Filtro aplicado: %s", dc.varFilter))
        dc.displayState(dc.currentState)
        return dc.readCommand()
    
    case "F", "nofilter", "quitarfiltro":
        dc.varFilter = ""
        dc.debugBuffer = append(dc.debugBuffer, "Filtro removido")
        dc.displayState(dc.currentState)
        return dc.readCommand()
    
    case "pgup":
        if dc.showGlobals {
            if dc.globalScroll > 0 {
                dc.globalScroll -= 5
                if dc.globalScroll < 0 {
                    dc.globalScroll = 0
                }
            }
        } else {
            if dc.localScroll > 0 {
                dc.localScroll -= 5
                if dc.localScroll < 0 {
                    dc.localScroll = 0
                }
            }
        }
        dc.displayState(dc.currentState)
        return dc.readCommand()
    
    case "pgdn":
        if dc.showGlobals {
            dc.globalScroll += 5
        } else {
            dc.localScroll += 5
        }
        dc.displayState(dc.currentState)
        return dc.readCommand()
    case "up":
        if dc.codeScroll > 0 {
            dc.codeScroll--
            dc.displayState(dc.currentState)
        }
        return dc.readCommand()
    
    case "down":
        dc.codeScroll++
        dc.displayState(dc.currentState)
        return dc.readCommand()
    
    case "quit", "q", "salir":
        dc.logger.Println("processCommand: Saliendo del debugger")
        os.Exit(0)
    
    default:
        dc.showError(fmt.Sprintf("Comando desconocido: %s", parts[0]))
        return dc.readCommand()
    }
    
    // No debería llegar aquí, pero por seguridad
    return dc.readCommand()
}

// evaluateExpression evalúa una expresión JavaScript en el contexto actual
func (dc *DebugConsole) evaluateExpression(expr string) {
    defer func() {
        if r := recover(); r != nil {
            dc.logger.Printf("evaluateExpression: Panic recovered: %v\n", r)
            dc.showError(fmt.Sprintf("Error: %v", r))
        }
    }()

    dc.logger.Printf("evaluateExpression: Evaluating: %s\n", expr)
    val, err := dc.runtime.RunString(expr)
    if err != nil {
        dc.logger.Printf("evaluateExpression: Error: %v\n", err)
        dc.showError(fmt.Sprintf("Error: %v", err))
        return
    }

    // Mostrar resultado en consola
    result := fmt.Sprintf("eval> %s = %s", expr, dc.formatValue(val))
    dc.consoleBuffer = append(dc.consoleBuffer, result)
    if len(dc.consoleBuffer) > dc.consoleMaxSize {
        dc.consoleBuffer = dc.consoleBuffer[1:]
    }
    
    // Redibujar para mostrar resultado
    dc.displayState(dc.currentState)
}

// min retorna el mínimo de dos enteros
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func main() {
    flag.Parse()
    args := flag.Args()

    if len(args) < 1 {
        fmt.Fprintf(os.Stderr, "Usage: %s <script.js>\n", os.Args[0])
        os.Exit(1)
    }

    dc := NewDebugConsole()
    defer dc.Close()

    if err := dc.LoadFile(args[0]); err != nil {
        fmt.Fprintf(os.Stderr, "Error loading file: %v\n", err)
        os.Exit(1)
    }

    if err := dc.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error running debugger: %v\n", err)
        os.Exit(1)
    }
}
```

## Archivo: run-debugger.sh

```bash
#!/bin/bash
# Script para ejecutar el debugger con la configuración correcta de terminal
# Este script soluciona problemas de compatibilidad con Enter y flechas

# Guardar configuración actual del terminal
OLD_STTY=$(stty -g)

# Función para restaurar la configuración del terminal al salir
cleanup() {
    stty "$OLD_STTY"
    echo ""
    echo "Terminal restaurado."
}

# Configurar trap para limpiar al salir
trap cleanup EXIT INT TERM

# Configurar el terminal para modo raw con echo
# Esto permite que las teclas funcionen correctamente
stty raw -echo

# Ejecutar el debugger
./goja-debug "$@"

# La limpieza se hace automáticamente por el trap
```

## Compilación e Instalación

```bash
# En el directorio goja-debug
go mod tidy
go build -o goja-debug

# Hacer ejecutable el script
chmod +x run-debugger.sh
```

## Uso

```bash
# Ejecutar con el script wrapper (recomendado)
./run-debugger.sh ejemplo_debug.js

# O directamente
./goja-debug ejemplo_debug.js
```

## Características Implementadas

1. **UI Completa**:
   - Ventana de código con resaltado de línea actual
   - Ventana de variables (locales/globales)
   - **Ventana de consola con dos columnas**: Salida del programa | Mensajes del debugger
   - Ventana de comandos con referencias actualizadas

2. **Navegación**:
   - Historial de comandos con flechas
   - **Repetición del último comando con Enter vacío**
   - Scroll en todas las ventanas
   - Filtrado de variables

3. **Debugging**:
   - Breakpoints
   - Step Into/Over/Out (con limitaciones documentadas en closures)
   - Evaluación de expresiones
   - Inspección de variables
   - **Stack trace con comando 'st' (anteriormente F5)**

4. **Manejo de Casos Especiales**:
   - Código interno del motor
   - Funciones nativas
   - Persistencia de línea actual
   - **Separación clara entre salida del programa y mensajes del debugger**

## Personalización

Para personalizar la consola:

1. **Colores**: Modificar las llamadas a `setColor`
2. **Layout**: Ajustar los cálculos de tamaño en `displayState`
3. **Comandos**: Agregar casos en `processCommand`
4. **Formato de variables**: Modificar `formatValue`

## Depuración del Debugger

El archivo `goja.debug.log` contiene información detallada de la ejecución. Útil para diagnosticar problemas con el debugger mismo.