// Demo script to show the new UI features
// This shows console output, variables, and nested calls

// Global variables
var appName = "Goja Debug UI Demo";
var version = 2.0;
var config = {
    theme: "dark",
    layout: "two-column",
    features: ["console", "variables", "callstack"]
};

// Function with local variables and console output
function processUser(user) {
    console.log("Processing user:", user.name);
    
    var fullName = user.firstName + " " + user.lastName;
    var age = calculateAge(user.birthYear);
    var status = age >= 18 ? "adult" : "minor";
    
    console.log("Full name:", fullName);
    console.log("Age:", age, "Status:", status);
    
    return {
        id: user.id,
        displayName: fullName,
        age: age,
        status: status
    };
}

// Helper function to demonstrate call stack
function calculateAge(birthYear) {
    var currentYear = new Date().getFullYear();
    var age = currentYear - birthYear;
    
    console.log("Calculating age for birth year", birthYear);
    
    return age;
}

// Main execution
console.log("=== Starting", appName, "v" + version, "===");
console.log("Config:", JSON.stringify(config));

var testUser = {
    id: 123,
    firstName: "John",
    lastName: "Doe",
    birthYear: 1990,
    name: "johndoe"
};

var result = processUser(testUser);
console.log("Result:", JSON.stringify(result));

// Array operations to test native functions
var numbers = [10, 20, 30, 40, 50];
var doubled = numbers.map(function(n) {
    console.log("Doubling:", n);
    return n * 2;
});

console.log("Original numbers:", numbers);
console.log("Doubled numbers:", doubled);

console.log("=== Demo Complete ===");