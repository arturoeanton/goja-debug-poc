// Final test for function stepping
console.log("1. Starting test");

function greet(name) {
    console.log("2. Inside greet function");
    return "Hello, " + name + "!";
}
let name1 = "Pedro";
console.log("3. Before calling greet");
let message;
if (name1 == "World") {
// Set breakpoint on this line
    message = greet("World");  
}else{
    message =  "Hello, " + name1 + "!";
}

console.log("4. After calling greet:", message);
console.log("5. Test completed");