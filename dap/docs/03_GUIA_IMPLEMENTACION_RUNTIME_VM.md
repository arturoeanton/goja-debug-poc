# Guía de Implementación del Debugger de Goja - Runtime y VM

## Modificaciones en runtime.go

### 1. Agregar campos al Runtime

En la estructura `Runtime` (aproximadamente línea 190), agregar:

```go
type Runtime struct {
    // ... campos existentes ...
    
    // Agregar estos campos:
    debugger       *Debugger
    enhancedErrors bool
    debugMode      bool
    
    // ... resto de campos ...
}
```

**Por qué**: 
- `debugger`: Referencia al debugger asociado con este runtime
- `debugMode`: Indica si el runtime está en modo debug (sin optimizaciones)
- `enhancedErrors`: Para errores mejorados (opcional pero útil)

### 2. Agregar RuntimeOptions

Antes de la función `New()` (aproximadamente línea 1350), agregar:

```go
// RuntimeOptions permite configurar el Runtime al crearlo
type RuntimeOptions struct {
    // EnableDebugMode activa el modo debug que desactiva optimizaciones
    EnableDebugMode bool
}
```

### 3. Modificar función New y agregar NewWithOptions

Agregar nueva función después de `New()`:

```go
// NewWithOptions crea un nuevo Runtime con opciones específicas
func NewWithOptions(opts RuntimeOptions) *Runtime {
    r := &Runtime{
        debugMode: opts.EnableDebugMode,
    }
    r.init()
    return r
}
```

### 4. Agregar método EnableDebugger

Después de los métodos de inicialización (aproximadamente línea 1400):

```go
// EnableDebugger activa el debugger para este runtime
func (r *Runtime) EnableDebugger() *Debugger {
    if r.debugger == nil {
        r.debugger = NewDebugger(r)
        r.debugger.Enable()
    }
    return r.debugger
}

// DisableDebugger desactiva el debugger
func (r *Runtime) DisableDebugger() {
    if r.debugger != nil {
        r.debugger.Disable()
    }
}

// GetDebugger retorna el debugger actual (puede ser nil)
func (r *Runtime) GetDebugger() *Debugger {
    return r.debugger
}

// IsDebugMode retorna true si el runtime está en modo debug
func (r *Runtime) IsDebugMode() bool {
    return r.debugMode
}
```

### 5. Modificar el método compile

Buscar el método `compile` (aproximadamente línea 1480) y modificar:

```go
func (r *Runtime) compile(name, src string, strict, inGlobal bool, evalVm *vm) (p *Program, err error) {
    // Cambiar esta línea:
    // p, err = compile(name, src, strict, inGlobal, evalVm, r.parserOptions...)
    
    // Por esta:
    p, err = compileWithDebugMode(name, src, strict, inGlobal, evalVm, r.debugMode, r.parserOptions...)
    
    // ... resto del método sin cambios ...
}
```

### 6. Agregar funciones de compilación con modo debug

Antes del método `compile`, agregar:

```go
func compileWithDebugMode(name, src string, strict, inGlobal bool, evalVm *vm, debugMode bool, parserOptions ...parser.Option) (p *Program, err error) {
    prg, err := Parse(name, src, parserOptions...)
    if err != nil {
        return
    }

    return compileASTWithDebugMode(prg, strict, inGlobal, evalVm, debugMode)
}

func compileASTWithDebugMode(prg *js_ast.Program, strict, inGlobal bool, evalVm *vm, debugMode bool) (p *Program, err error) {
    c := newCompilerWithDebugMode(debugMode)

    defer func() {
        if x := recover(); x != nil {
            p = nil
            switch x1 := x.(type) {
            case *CompilerSyntaxError:
                err = x1
            case *CompilerReferenceError:
                err = x1
            default:
                panic(x)
            }
        }
    }()

    if evalVm != nil {
        c.ctxVM = evalVm
    }

    c.compile(prg, strict, inGlobal, evalVm)
    p = c.p
    return
}
```

## Modificaciones en vm.go

### 1. Agregar hooks de debugging en el método run

Buscar el método `run` de la VM (aproximadamente línea 600) y modificar el loop principal:

```go
func (vm *vm) run() {
    // ... código existente ...
    
    for !vm.halt {
        // Agregar hook de debugging ANTES de ejecutar la instrucción
        if vm.r.debugger != nil && vm.r.debugger.enabled {
            if vm.r.debugger.checkBreakpoint(vm) {
                // Log opcional
                if vm.r.debugger.logger != nil {
                    vm.r.debugger.logger.Printf("VM: Pausing at PC=%d\n", vm.pc)
                }
                vm.r.debugger.handlePause(vm)
            }
        }
        
        // Código existente de ejecución de instrucción
        if vm.halted() {
            break
        }
        
        vm.prg.code[vm.pc].exec(vm)
        ticks++
        
        // ... resto del código sin cambios ...
    }
}
```

**Por qué**: Este es el punto donde la VM verifica antes de cada instrucción si debe pausar por un breakpoint o step.

### 2. Agregar método para obtener información de debugging

Al final del archivo vm.go, agregar:

```go
// debugInfo retorna información de debugging de la VM
func (vm *vm) debugInfo() (pc int, prg *Program, callStackLen int) {
    return vm.pc, vm.prg, len(vm.callStack)
}

// currentPosition retorna la posición actual en el código fuente
func (vm *vm) currentPosition() (position file.Position, ok bool) {
    if vm.prg != nil && vm.pc < len(vm.prg.src) {
        if pos := vm.prg.src[vm.pc]; pos.srcPos > 0 {
            position = vm.prg.Position(pos.srcPos)
            ok = true
        }
    }
    return
}

// isInNativeCall verifica si estamos en una llamada nativa
func (vm *vm) isInNativeCall() bool {
    // Verificar si el frame actual es una función nativa
    if len(vm.callStack) > 0 {
        frame := &vm.callStack[len(vm.callStack)-1]
        if frame.prg == nil {
            return true
        }
    }
    return false
}
```

### 3. Exportar campos necesarios (si son privados)

Si algunos campos de la VM son privados y necesitas accederlos desde el debugger, puedes:

1. Hacerlos públicos (cambiar primera letra a mayúscula)
2. O crear métodos getter

Por ejemplo:

```go
// En vm.go, agregar getters si es necesario:
func (vm *vm) Stack() []Value {
    return vm.stack
}

func (vm *vm) PC() int {
    return vm.pc
}

func (vm *vm) CallStack() []frame {
    return vm.callStack
}

func (vm *vm) Stash() *stash {
    return vm.stash
}

func (vm *vm) SB() int {
    return vm.sb
}
```

## Modificaciones en frame (si es necesario)

Si la estructura `frame` en vm.go es privada, agregar métodos para acceder a su información:

```go
// Agregar estos métodos a la estructura frame
func (f *frame) FuncName() string {
    if f.prg != nil && f.prg.funcName != "" {
        return f.prg.funcName.String()
    }
    return "<anonymous>"
}

func (f *frame) Position() string {
    if f.prg != nil && f.pc < len(f.prg.src) {
        if pos := f.prg.src[f.pc]; pos.srcPos > 0 {
            position := f.prg.Position(pos.srcPos)
            return fmt.Sprintf("%s:%d:%d", position.Filename, position.Line, position.Column)
        }
    }
    return "<unknown>"
}
```

## Consideraciones de Rendimiento

1. **Check de debugger**: El check `if vm.r.debugger != nil && vm.r.debugger.enabled` es muy rápido cuando el debugger no está activo.

2. **Optimización**: El método `checkBreakpoint` debe ser eficiente ya que se llama en cada instrucción.

3. **Modo debug**: Cuando `debugMode` está activo, el compilador generará código menos optimizado pero más debuggeable.

## Testing

Para verificar la integración:

```go
func TestDebuggerIntegration(t *testing.T) {
    // Crear runtime con debug mode
    opts := RuntimeOptions{
        EnableDebugMode: true,
    }
    r := NewWithOptions(opts)
    
    // Habilitar debugger
    debugger := r.EnableDebugger()
    
    // Establecer un handler simple
    debugger.SetHandler(func(state *DebuggerState) DebugCommand {
        t.Logf("Paused at line %d\n", state.SourcePos.Line)
        return DebugContinue
    })
    
    // Agregar breakpoint
    debugger.AddBreakpoint("test.js", 2, 0)
    
    // Ejecutar código
    _, err := r.RunString(`
        var x = 1;
        x = x + 1; // Línea 2 - breakpoint aquí
        console.log(x);
    `)
    
    if err != nil {
        t.Fatal(err)
    }
}
```

## Siguiente Paso

Continúa con [04_GUIA_IMPLEMENTACION_COMPILER.md](04_GUIA_IMPLEMENTACION_COMPILER.md) para implementar el modo debug en el compilador.