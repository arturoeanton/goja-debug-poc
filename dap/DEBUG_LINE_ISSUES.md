# Depuración de Problemas de Líneas

## Cómo diagnosticar los problemas

### 1. Problema del Off-by-One (para una línea antes)

Para depurar este problema:

1. Inicia el servidor con logs: `./debug-server.sh`
2. Pon un breakpoint en una línea específica (ej: línea 4)
3. Observa los logs cuando se agregue el breakpoint:
   ```
   Added breakpoint: file=test.js, line=4, column=0, gojaID=1, dapID=1
   ```
4. Cuando se detenga, verifica si coincide:
   ```
   Hit breakpoint ID=1 at test.js:4:0 (expected line 4)
   ```

Si dice que para en línea 3 cuando esperabas línea 4, hay un problema de offset.

### 2. Problema del If-Else (pasa por ambas ramas)

Los logs mejorados ahora muestran el contenido de cada línea:

```
Stepped to test.js:6:0 (PC=15) | if (sum > 10) {
Stepped to test.js:7:4 (PC=20) | console.log("The sum is greater than 10.");
Stepped to test.js:8:0 (PC=25) | } else {
Stepped to test.js:11:0 (PC=30) | }
```

Nota cómo pasa por la línea 8 (else) aunque no ejecuta el código dentro.

## Posibles causas

### Off-by-One:
1. **Diferencia en indexación**: VS Code usa 1-based, algunos sistemas usan 0-based
2. **Source mapping**: El compilador puede mapear instrucciones a líneas diferentes
3. **Carácteres invisibles**: Newlines extras o caracteres especiales

### If-Else:
1. **AST Structure**: El parser trata if-else como una unidad
2. **Bytecode generation**: Las instrucciones de salto incluyen referencias a ambas ramas
3. **Debug info**: La información de debug incluye todas las posiciones del statement

## Soluciones temporales

1. **Para Off-by-One**: 
   - Verifica que no haya líneas en blanco o caracteres extraños
   - Usa el archivo test-lines.js para verificar el comportamiento

2. **Para If-Else**:
   - Usa breakpoints dentro de los bloques específicos
   - Ignora los pasos por el else cuando no se ejecute
   - Usa Continue (F5) en lugar de Step Over (F10)

## Logs detallados

El servidor ahora proporciona:
- Número de línea y columna exactos
- Program Counter (PC)
- Contenido de la línea actual
- IDs de breakpoints esperados vs actuales

Esto ayuda a diagnosticar exactamente qué está pasando.