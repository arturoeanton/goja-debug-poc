# Guía de Implementación del Debugger de Goja - API Pública

## API del Runtime

### RuntimeOptions

```go
// RuntimeOptions configura el comportamiento del Runtime
type RuntimeOptions struct {
    // EnableDebugMode activa el modo debug (sin optimizaciones)
    EnableDebugMode bool
}
```

### Métodos del Runtime

```go
// Crear runtime con opciones
runtime := goja.NewWithOptions(goja.RuntimeOptions{
    EnableDebugMode: true,
})

// Habilitar debugger
debugger := runtime.EnableDebugger()

// Deshabilitar debugger
runtime.DisableDebugger()

// Obtener debugger (puede ser nil)
debugger := runtime.GetDebugger()

// Verificar si está en modo debug
if runtime.IsDebugMode() {
    // El compilador no optimizará el código
}
```

## API del Debugger

### Tipos Principales

```go
// DebugCommand - Comandos de control
type DebugCommand int

const (
    DebugContinue DebugCommand = iota  // Continuar ejecución
    DebugStepOver                      // Siguiente línea (sin entrar en funciones)
    DebugStepInto                      // Siguiente línea (entrando en funciones)
    DebugStepOut                       // Salir de la función actual
    DebugPause                         // Pausar en la próxima oportunidad
)

// DebugHandler - Callback cuando el debugger se pausa
type DebugHandler func(state *DebuggerState) DebugCommand

// DebuggerState - Estado cuando el debugger está pausado
type DebuggerState struct {
    PC                 int             // Program Counter
    SourcePos          file.Position   // Posición en el código fuente
    Breakpoint         *Breakpoint     // Breakpoint actual (si aplica)
    StepMode           bool            // Si está en modo stepping
    InNativeCall       bool            // Si está en una función nativa
    NativeFunctionName string          // Nombre de la función nativa
    CallStack          []StackFrame    // Stack de llamadas
    DebugStack         []DebugFrame    // Stack con info de debugging
}

// StackFrame - Frame en el call stack
type StackFrame interface {
    FuncName() string    // Nombre de la función
    Position() string    // Posición en formato "archivo:línea:columna"
}

// DebugFrame - Frame con información detallada
type DebugFrame struct {
    ID         int           // ID del frame (0 = actual)
    Name       string        // Nombre de la función
    SourcePos  file.Position // Posición en el código
    This       Value         // Valor de 'this'
    Scopes     []Scope       // Ámbitos de variables
}

// Scope - Ámbito de variables
type Scope struct {
    Name          string  // "Local", "Closure", "Global"
    VariablesRef  int     // Referencia para obtener variables
    NamedVars     int     // Número de variables con nombre
    IndexedVars   int     // Número de variables indexadas
}

// Variable - Representa una variable
type Variable struct {
    Name           string  // Nombre de la variable
    Value          Value   // Valor actual
    Type           string  // Tipo de la variable
    VariablesRef   int     // Ref si el valor tiene sub-valores
    IndexedVars    int     // Número de elementos (arrays)
    NamedVars      int     // Número de propiedades (objetos)
}

// Breakpoint - Punto de interrupción
type Breakpoint struct {
    // Métodos públicos:
    ID() int                    // ID único del breakpoint
    SourcePos() file.Position   // Posición del breakpoint
    Enable()                    // Habilitar breakpoint
    Disable()                   // Deshabilitar breakpoint
    IsEnabled() bool           // Estado del breakpoint
    Hits() int                 // Veces que se ha alcanzado
}
```

### Métodos del Debugger

#### Control de Ejecución

```go
// Establecer handler para eventos de debugging
debugger.SetHandler(func(state *DebuggerState) DebugCommand {
    // Manejar el evento de pausa
    fmt.Printf("Pausado en línea %d\n", state.SourcePos.Line)
    
    // Retornar comando para continuar
    return DebugContinue
})

// Continuar ejecución
debugger.Continue()

// Step over (siguiente línea sin entrar en funciones)
debugger.StepOver()

// Step into (siguiente línea entrando en funciones)
debugger.StepInto()

// Step out (salir de la función actual)
debugger.StepOut()

// Pausar ejecución
debugger.Pause()
```

#### Gestión de Breakpoints

```go
// Agregar breakpoint
id := debugger.AddBreakpoint("script.js", 10, 0) // archivo, línea, columna

// Remover breakpoint
removed := debugger.RemoveBreakpoint(id)

// Limpiar todos los breakpoints
debugger.ClearBreakpoints()

// Trabajar con un breakpoint
if state.Breakpoint != nil {
    bp := state.Breakpoint
    fmt.Printf("Hit breakpoint #%d at line %d\n", bp.ID(), bp.SourcePos().Line)
    
    // Deshabilitar temporalmente
    bp.Disable()
    
    // Volver a habilitar
    bp.Enable()
    
    // Ver cuántas veces se ha alcanzado
    fmt.Printf("Hit count: %d\n", bp.Hits())
}
```

#### Inspección de Variables

```go
// En el handler, obtener variables de los scopes
for _, frame := range state.DebugStack {
    fmt.Printf("Frame: %s at %s\n", frame.Name, frame.SourcePos)
    
    for _, scope := range frame.Scopes {
        fmt.Printf("  Scope: %s\n", scope.Name)
        
        // Obtener variables del scope
        vars := debugger.GetVariables(scope.VariablesRef)
        
        for _, v := range vars {
            fmt.Printf("    %s = %v (%s)\n", v.Name, v.Value, v.Type)
            
            // Si es un objeto/array, puede tener sub-valores
            if v.VariablesRef > 0 {
                subVars := debugger.GetVariables(v.VariablesRef)
                // ... procesar sub-valores ...
            }
        }
    }
}
```

#### Utilidades

```go
// Establecer logger para debugging del debugger
logger := log.New(os.Stdout, "[DEBUG] ", log.LstdFlags)
debugger.SetLogger(logger)
```

## Ejemplo Completo de Uso

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/dop251/goja"
)

func main() {
    // Crear runtime con modo debug
    opts := goja.RuntimeOptions{
        EnableDebugMode: true,
    }
    runtime := goja.NewWithOptions(opts)
    
    // Habilitar debugger
    debugger := runtime.EnableDebugger()
    
    // Configurar logger (opcional)
    debugger.SetLogger(log.New(os.Stdout, "[DEBUGGER] ", log.LstdFlags))
    
    // Variable para controlar el debugging
    stepMode := false
    
    // Establecer handler
    debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
        // Mostrar información del estado
        fmt.Printf("\n=== PAUSADO ===\n")
        fmt.Printf("Archivo: %s\n", state.SourcePos.Filename)
        fmt.Printf("Línea: %d, Columna: %d\n", state.SourcePos.Line, state.SourcePos.Column)
        
        // Si es un breakpoint
        if state.Breakpoint != nil {
            fmt.Printf("Breakpoint #%d (hits: %d)\n", 
                state.Breakpoint.ID(), 
                state.Breakpoint.Hits())
        }
        
        // Mostrar call stack
        fmt.Println("\nCall Stack:")
        for i, frame := range state.CallStack {
            fmt.Printf("  %d: %s at %s\n", i, frame.FuncName(), frame.Position())
        }
        
        // Mostrar variables locales
        if len(state.DebugStack) > 0 {
            fmt.Println("\nVariables:")
            frame := state.DebugStack[0]
            
            for _, scope := range frame.Scopes {
                if scope.Name == "Local" {
                    vars := debugger.GetVariables(scope.VariablesRef)
                    for _, v := range vars {
                        fmt.Printf("  %s = %v\n", v.Name, v.Value)
                    }
                }
            }
        }
        
        // Decidir qué hacer
        if stepMode {
            return goja.DebugStepOver
        } else {
            return goja.DebugContinue
        }
    })
    
    // Agregar algunos breakpoints
    debugger.AddBreakpoint("test.js", 3, 0)
    debugger.AddBreakpoint("test.js", 7, 0)
    
    // Código JavaScript a ejecutar
    script := `
        function factorial(n) {
            if (n <= 1) {          // Línea 3 - breakpoint
                return 1;
            }
            
            let result = n * factorial(n - 1);  // Línea 7 - breakpoint
            return result;
        }
        
        console.log("Factorial de 5:", factorial(5));
    `
    
    // Configurar console.log
    console := runtime.NewObject()
    console.Set("log", func(args ...interface{}) {
        fmt.Println("[CONSOLE]", args...)
    })
    runtime.Set("console", console)
    
    // Ejecutar el script
    _, err := runtime.RunString(script)
    if err != nil {
        panic(err)
    }
}
```

## Casos de Uso Avanzados

### 1. Debugging Condicional

```go
debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
    // Solo pausar si una variable tiene cierto valor
    for _, frame := range state.DebugStack {
        for _, scope := range frame.Scopes {
            vars := debugger.GetVariables(scope.VariablesRef)
            for _, v := range vars {
                if v.Name == "counter" && v.Value.ToInteger() > 100 {
                    fmt.Println("Counter exceeded 100!")
                    return goja.DebugPause
                }
            }
        }
    }
    return goja.DebugContinue
})
```

### 2. Logging de Ejecución

```go
var executionLog []string

debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
    // Registrar cada línea ejecutada
    logEntry := fmt.Sprintf("%s:%d", 
        state.SourcePos.Filename, 
        state.SourcePos.Line)
    executionLog = append(executionLog, logEntry)
    
    // Continuar sin pausar
    return goja.DebugContinue
})

// Activar step mode para registrar todo
debugger.StepInto()
```

### 3. Breakpoints Dinámicos

```go
debugger.SetHandler(func(state *goja.DebuggerState) goja.DebugCommand {
    // Agregar breakpoint cuando se encuentra cierta función
    for _, frame := range state.CallStack {
        if frame.FuncName() == "problematicFunction" {
            // Agregar breakpoint en la siguiente línea
            debugger.AddBreakpoint(
                state.SourcePos.Filename,
                state.SourcePos.Line + 1,
                0,
            )
        }
    }
    return goja.DebugContinue
})
```

## Limitaciones y Consideraciones

1. **Thread Safety**: El debugger NO es thread-safe. Usar un debugger por runtime.

2. **Performance**: El modo debug impacta significativamente el rendimiento.

3. **Closures**: Step-into puede no funcionar automáticamente con closures.

4. **Código Nativo**: Las funciones nativas de Go no pueden ser debuggeadas línea por línea.

5. **Eval**: El código ejecutado con eval() tiene soporte limitado.

## Siguiente Paso

Continúa con [06_GUIA_IMPLEMENTACION_EJEMPLO_CONSOLA.md](06_GUIA_IMPLEMENTACION_EJEMPLO_CONSOLA.md) para ver cómo implementar una consola de debugging completa.