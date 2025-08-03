# Guía de Implementación del Debugger de Goja - Compilador con Modo Debug

## Modificaciones en compiler.go

### 1. Agregar campo debugMode a la estructura compiler

En la estructura `compiler` (aproximadamente línea 77), agregar:

```go
type compiler struct {
    p     *Program
    scope *scope
    block *block
    
    // ... otros campos ...
    
    // Agregar este campo:
    debugMode bool // Force all variables to stash for debugging
}
```

**Por qué**: En modo debug, necesitamos forzar todas las variables al stash para que sean inspeccionables, evitando optimizaciones que las pondrían en el stack.

### 2. Modificar newCompiler para aceptar debugMode

Buscar la función `newCompiler` y crear una nueva versión:

```go
// Mantener la función original para compatibilidad
func newCompiler() *compiler {
    return newCompilerWithDebugMode(false)
}

// Nueva función con soporte para debug mode
func newCompilerWithDebugMode(debugMode bool) *compiler {
    c := &compiler{
        p:         &Program{},
        debugMode: debugMode,
    }
    
    c.enumGetExpr.init(c, file.Idx(0))
    c.scope = &scope{c: c}
    c.block = &block{c: c}
    return c
}
```

### 3. Modificar binding.moveToStash

Buscar el método `moveToStash` de la estructura `binding` (aproximadamente línea 260):

```go
func (b *binding) moveToStash() {
    if b.isArg && !b.scope.argsInStash {
        b.scope.moveArgsToStash()
    } else {
        b.inStash = true
        b.scope.needStash = true
    }
}
```

**Nota**: Este método ya existe, no necesita modificación, pero es importante entender que se usa para mover variables al stash.

### 4. Modificar métodos de creación de bindings

#### a) En createVarIdBinding (aproximadamente línea 1100):

```go
func (c *compiler) createVarIdBinding(name unistring.String, offset int) *binding {
    // ... código existente ...
    
    if !inFunc || name != "arguments" {
        b, _ := c.scope.bindName(name)
        // Agregar esta verificación:
        if c.debugMode && b != nil {
            b.moveToStash()
        }
        return b
    }
    
    // ... resto del código ...
}
```

#### b) En createLexicalIdBinding (aproximadamente línea 1170):

```go
func (c *compiler) createLexicalIdBinding(name unistring.String, isConst bool, offset int) *binding {
    // ... código existente ...
    
    // Al final del método, antes del return:
    if c.debugMode && b != nil {
        b.moveToStash()
    }
    return b
}
```

#### c) En createPatternLexicalBinding (aproximadamente línea 1200):

```go
func (c *compiler) createPatternLexicalBinding(name unistring.String, isConst bool, offset int) *binding {
    // ... código existente ...
    
    // Al final del método, antes del return:
    if c.debugMode && b != nil {
        b.moveToStash()
    }
    return b
}
```

### 5. Modificar finaliseVarAlloc

Buscar el método `finaliseVarAlloc` en la estructura `scope` (aproximadamente línea 620):

```go
func (s *scope) finaliseVarAlloc(stackOffset int) (stashSize, stackSize int) {
    // ... código inicial ...
    
    // Cambiar esta línea:
    // allInStash := s.isDynamic()
    
    // Por esta:
    allInStash := s.isDynamic() || s.c.debugMode
    
    // ... resto del método sin cambios ...
}
```

**Por qué**: Esto fuerza todas las variables al stash cuando estamos en modo debug, haciéndolas accesibles para inspección.

## Modificaciones en compiler_expr.go

### 1. Prevenir uso de enterFuncStashless en modo debug

Buscar la función `emitFunction` (aproximadamente línea 1700) y modificar la condición que decide qué tipo de función crear:

```go
// Buscar esta condición (aproximadamente línea 1723):
if stashSize > 0 || s.argsInStash {

// Cambiarla por:
if stashSize > 0 || s.argsInStash || e.c.debugMode {
```

**Por qué**: `enterFuncStashless` es una versión optimizada que no soporta debugging completo. En modo debug, siempre usamos la versión completa.

### 2. Forzar argumentos al stash en modo debug

En la misma función `emitFunction`, buscar (aproximadamente línea 1650):

```go
// Buscar:
if s.isDynamic() && !s.argsInStash {
    s.moveArgsToStash()
}

// Cambiar por:
if (s.isDynamic() || e.c.debugMode) && !s.argsInStash {
    s.moveArgsToStash()
}
```

### 3. Asegurar creación del objeto 'arguments'

Unas líneas más abajo:

```go
// Buscar:
if s.argsNeeded || s.isDynamic() && e.typ != funcArrow && e.typ != funcClsInit {

// Cambiar por:
if s.argsNeeded || (s.isDynamic() || e.c.debugMode) && e.typ != funcArrow && e.typ != funcClsInit {
```

### 4. Manejar forward references en modo debug

```go
// Buscar (en el manejo de parámetros con inicializadores):
if firstForwardRef == -1 && (s.isDynamic() || s.bindings[i].useCount() > 0) {

// Cambiar por:
if firstForwardRef == -1 && (s.isDynamic() || e.c.debugMode || s.bindings[i].useCount() > 0) {
```

### 5. Prevenir eliminación de bindings no usados

```go
// Buscar (manejo de calleeBinding):
if !s.isDynamic() && calleeBinding.useCount() == 0 {
    s.deleteBinding(calleeBinding)
    calleeBinding = nil
}

// Cambiar por:
if !s.isDynamic() && !e.c.debugMode && calleeBinding.useCount() == 0 {
    s.deleteBinding(calleeBinding)
    calleeBinding = nil
}
```

Similar para thisBinding:

```go
// Buscar:
if !s.isDynamic() && thisBinding.useCount() == 0 {

// Cambiar por:
if !s.isDynamic() && !e.c.debugMode && thisBinding.useCount() == 0 {
```

### 6. Forzar this al stash en modo debug

```go
// Buscar:
if thisBinding.inStash || s.isDynamic() {

// Cambiar por:
if thisBinding.inStash || s.isDynamic() || e.c.debugMode {
```

### 7. Nombres de funciones en modo debug

En lugares donde se verifica `s.isDynamic()` para incluir nombres:

```go
// Buscar patrones como:
if s.isDynamic() || e.c.debugMode {
    enter1.names = s.makeNamesMap()
}
```

Asegurarse de que en modo debug siempre se incluyan los nombres de las variables.

### 8. Manejo de constructores de clase

Para constructores y métodos de clase:

```go
// En compileFunctionLiteral, buscar:
if s.isDynamic() || e.c.debugMode || thisBinding.useCount() > 0 {
    if s.isDynamic() || e.c.debugMode || thisBinding.inStash {
        thisBinding.emitInitAt(1)
    }
}
```

## Resumen de Cambios

Los cambios en el compilador tienen tres objetivos principales:

1. **Forzar variables al stash**: En modo debug, todas las variables van al stash (un mapa) en lugar del stack (array), permitiendo inspección por nombre.

2. **Prevenir optimizaciones**: 
   - No usar `enterFuncStashless`
   - No eliminar bindings "no usados"
   - Mantener toda la información de nombres

3. **Preservar información**: 
   - Mantener nombres de variables
   - Mantener referencias a this y arguments
   - No optimizar closures

## Testing del Modo Debug

```go
func TestDebugModeCompilation(t *testing.T) {
    // Compilar con modo debug
    prg, err := compileWithDebugMode("test.js", `
        function test(a, b) {
            var x = a + b;
            let y = x * 2;
            const z = y + 1;
            return z;
        }
    `, false, true, nil, true) // true = debugMode
    
    if err != nil {
        t.Fatal(err)
    }
    
    // Verificar que se generó código debug-friendly
    // Por ejemplo, verificar que no hay instrucciones stashless
    for _, instr := range prg.code {
        if _, ok := instr.(*enterFuncStashless); ok {
            t.Error("Found stashless function in debug mode")
        }
    }
}
```

## Impacto en Rendimiento

**IMPORTANTE**: El modo debug tiene un impacto significativo en el rendimiento:
- Todas las variables en stash = accesos más lentos
- Sin optimizaciones = más instrucciones
- Información adicional = más memoria

**Recomendación**: Usar modo debug SOLO durante desarrollo/debugging, nunca en producción.

## Siguiente Paso

Continúa con [05_GUIA_IMPLEMENTACION_API.md](05_GUIA_IMPLEMENTACION_API.md) para ver la API pública completa del debugger.