// Very simple test for step-into
var x = 5;

function add(a, b) {
    var result = a + b;
    console.log("Adding:", a, "+", b, "=", result);
    return result;
}

console.log("Before function call");
var sum = add(x, 3);  // Try step-into here
console.log("After function call, sum =", sum);