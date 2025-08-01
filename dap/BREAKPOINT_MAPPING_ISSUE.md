# Problema de Mapeo de Breakpoints

## Descripción del Problema

Los breakpoints no paran en la línea correcta. Específicamente:

1. **Breakpoint en línea 3** (`var sum = x + y;`) - No para
2. **Breakpoints en líneas 3 y 4** - Para en línea 2 (`var y = 10;`)

## Posibles Causas

### 1. Desajuste de Nombres de Archivo
VS Code puede enviar paths absolutos mientras que Goja usa paths relativos:
- VS Code: `/Users/.../test.js`
- Goja: `test.js`

**Solución implementada**: Normalizamos el filename para que coincida.

### 2. Mapeo de Instrucciones
El compilador de Goja puede estar mapeando las instrucciones a líneas diferentes:
- La instrucción para `var sum = x + y` puede estar asociada con la línea anterior
- Esto es común en compiladores que optimizan o reorganizan código

### 3. Problema de Source Positions
Goja genera posiciones de source basándose en el AST, que puede no coincidir exactamente con las líneas del código fuente.

## Cómo Diagnosticar

1. **Inicia con logs detallados**:
   ```bash
   ./debug-server.sh
   ```

2. **Observa los logs al poner breakpoints**:
   ```
   SetBreakpoints request for file: test.js, breakpoints: 1
   Added breakpoint: file=test.js, line=3, column=0, gojaID=1, dapID=1
   ```

3. **Observa cuando "debería" parar**:
   ```
   DEBUG: At breakpoint line 3 but state.Breakpoint is nil!
   ```

4. **Observa dónde realmente para**:
   ```
   Hit breakpoint ID=1 at test.js:2:0 (BP was set for line 3)
   WARNING: Breakpoint line mismatch! Stopped at line 2 but breakpoint is for line 3
   ```

## Workarounds

### 1. Ajustar Breakpoints
Si sabes que para una línea antes, pon el breakpoint una línea después:
- Quieres parar en línea 3 → Pon breakpoint en línea 4

### 2. Usar Múltiples Breakpoints
Pon breakpoints en varias líneas consecutivas para asegurar que pare.

### 3. Usar console.log
Agrega `console.log` para confirmar dónde está ejecutando realmente.

## Solución Definitiva

Este problema requiere cambios en el motor de Goja para:
1. Mejorar el mapeo de source positions
2. Asociar las instrucciones con las líneas correctas
3. Posiblemente ajustar cómo se generan las posiciones en el compilador

## Test Case

Usa `test-bp.js` para reproducir el problema:
```javascript
// Line 1 - comment
var x = 5;              // Line 2
var y = 10;             // Line 3
var sum = x + y;        // Line 4 - Put BP here
console.log("Sum:", sum); // Line 5
```