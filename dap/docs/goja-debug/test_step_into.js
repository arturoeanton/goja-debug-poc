// Test script to verify step-into functionality

function outer() {
    console.log("In outer function");
    var result = inner(5); // Try to step into this
    console.log("Back in outer, result:", result);
    return result;
}

function inner(x) {
    console.log("In inner function with x =", x);
    var doubled = x * 2;
    return doubled;
}

// Test different function call scenarios
console.log("Starting step-into test");

// Regular function call
var test1 = outer();

// Function expression
var multiply = function(a, b) {
    console.log("Multiplying", a, "and", b);
    return a * b;
};
var test2 = multiply(3, 4); // Try to step into this

// Arrow function (if supported)
var add = (a, b) => {
    console.log("Adding", a, "and", b);
    return a + b;
};
var test3 = add(10, 20); // Try to step into this

console.log("Tests complete:", test1, test2, test3);