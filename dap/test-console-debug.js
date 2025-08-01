// Test console.log debugging
var x = 5;
console.log("Before function");

function testFunc(param) {
    var local = param * 2;
    console.log("Inside function, param:", param, "local:", local);
    return local;
}

var result = testFunc(x);
console.log("After function, result:", result);