# Local Variables Inspection Issue

## Current Status

The DAP server has been updated to fix two issues:

1. **Console.log with multiple arguments** - FIXED
   - Changed from variadic Go function to proper Goja function with FunctionCall
   - Now correctly handles multiple arguments passed to console.log

2. **Nested object/array inspection** - FIXED
   - Objects and arrays now show expandable references in the debugger
   - You can click to inspect nested properties

## Remaining Issue: Local Variables in Functions

The current implementation cannot show local variables inside functions (like the `message` parameter in `printMessage`). This is a limitation of the current Goja debugger API.

### Why Local Variables Don't Show

1. The Goja debugger provides stack frames but doesn't expose the scope chain or local variables for each frame
2. We can only access the global object, not the local scope of functions
3. The `goja.StackFrame` type doesn't have methods to access local variables

### What Would Be Needed

To properly support local variable inspection, the Goja debugger API would need to be enhanced:

1. **Expose Scope Chain**: Add methods to `goja.StackFrame` to access the scope chain
   ```go
   type StackFrame struct {
       // ... existing fields
   }
   
   // Needed methods:
   func (f *StackFrame) GetLocalScope() *Object
   func (f *StackFrame) GetClosureScope() *Object
   func (f *StackFrame) GetParameters() map[string]Value
   ```

2. **Access Function Parameters**: Provide access to function parameters and their values
3. **Access Local Variables**: Provide access to variables declared within the function

### Workaround

For now, you can use the debugger's evaluate feature to inspect local variables:
- When paused inside a function, use the Debug Console to evaluate expressions
- Type the variable name (e.g., `message`) to see its value
- This works because the evaluation happens in the current scope

### Next Steps

To fully fix this issue, we would need to:
1. Fork and modify the Goja runtime to expose local scopes in the debugger API
2. Update the DAP adapter to use these new APIs
3. Test thoroughly to ensure all scope types are handled correctly

The console.log issue has been fixed and nested object inspection now works properly.