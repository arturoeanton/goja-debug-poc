#!/bin/bash

# Script para demostrar las capacidades del debugger de Goja

echo "🐛 GOJA Debug Console Demo"
echo "========================="
echo ""
echo "Este demo muestra todas las características del debugger:"
echo "- Visualización de código con línea actual"
echo "- Variables locales y globales"
echo "- Call stack"
echo "- REPL interactivo"
echo "- Step over/into/out"
echo "- Breakpoints"
echo "- Manejo de funciones nativas"
echo ""
echo "Presiona Enter para comenzar..."
read

# Ejecutar el debugger con el archivo de prueba
./goja-debug test_debug.js