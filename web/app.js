const baseUrl = "http://localhost:8080";
let token = "";

async function login() {
    const response = await fetch(`${baseUrl}/api/v1/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            email: document.getElementById("email").value,
            password: document.getElementById("password").value
        })
    });

    const data = await response.json();
    token = data.token || "";
    document.getElementById("loginResult").textContent = JSON.stringify(data, null, 2);
}

async function loadProducts() {
    const response = await fetch(`${baseUrl}/api/v1/products`);
    const data = await response.json();
    document.getElementById("products").textContent = JSON.stringify(data, null, 2);
}

function connectNotifications() {
    if (!token) {
        document.getElementById("notifications").textContent = "Login first.";
        return;
    }

    const socket = new WebSocket(`ws://localhost:8080/ws/notifications?token=${encodeURIComponent(token)}`);
    socket.onopen = () => {
        document.getElementById("notifications").textContent = "WebSocket connected.\n";
    };
    socket.onmessage = (event) => {
        const element = document.getElementById("notifications");
        element.textContent += event.data + "\n";
    };
    socket.onclose = () => {
        document.getElementById("notifications").textContent += "WebSocket closed.\n";
    };
}
