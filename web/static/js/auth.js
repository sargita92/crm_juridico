function togglePassword() {
    var input = document.getElementById("password");
    var eyeOn = document.getElementById("eye-icon");
    var eyeOff = document.getElementById("eye-off-icon");

    if (input.type === "password") {
        input.type = "text";
        eyeOn.style.display = "none";
        eyeOff.style.display = "block";
    } else {
        input.type = "password";
        eyeOn.style.display = "block";
        eyeOff.style.display = "none";
    }
}

// Re-attach toggle after HTMX swap (login card re-render on error)
document.addEventListener("htmx:afterSwap", function(event) {
    if (event.detail.target.id === "login-card") {
        var emailInput = document.getElementById("email");
        if (emailInput) emailInput.focus();
    }
});
