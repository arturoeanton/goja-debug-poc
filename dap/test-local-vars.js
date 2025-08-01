// Global variables
var globalX = 100;
var globalY = 200;

function testFunction(param1, param2) {
    // Local variables
    var localA = param1 + 10;
    var localB = param2 * 2;
    
    // Set a breakpoint here and check variables
    console.log("Inside function:");
    console.log("param1 =", param1);
    console.log("param2 =", param2);
    console.log("localA =", localA);
    console.log("localB =", localB);
    
    return localA + localB;
}

// Call the function
var result = testFunction(5, 7);
console.log("Result =", result);
console.log("GlobalX =", globalX);
console.log("GlobalY =", globalY);