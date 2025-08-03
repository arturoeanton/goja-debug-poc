// Test script to verify all variables are accessible in debug mode
function testFunction(param1, param2) {
    var localVar1 = 10;
    let localVar2 = 20;
    const localVar3 = 30;
    
    // Set a breakpoint here to inspect all variables
    console.log("Breakpoint location - all variables should be visible");
    
    // Nested function to test closure variables
    function innerFunction() {
        var innerVar = 40;
        console.log("Inner function - should see closure variables");
        return param1 + param2 + localVar1 + localVar2 + localVar3 + innerVar;
    }
    
    return innerFunction();
}

// Global variables
var globalVar1 = 100;
let globalVar2 = 200;
const globalVar3 = 300;

// Call the function
var result = testFunction(5, 15);
console.log("Result:", result);