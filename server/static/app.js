const agentsList = document.getElementById('agents');
const commandInput = document.getElementById('command');

const socket = new WebSocket(`ws://${location.host}/ws-dashboard`);

socket.onopen = () => {
    console.log('Connected to dashboard websocket');
};

socket.onmessage = (event) => {
    // For now, we'll just log messages from the server
    console.log('Message from server:', event.data);

    // In a real application, you would update the agent list here
    // For this basic example, we'll just add a placeholder
    const agentId = `Agent-${Math.floor(Math.random() * 1000)}`;
    const newAgent = document.createElement('li');
    newAgent.textContent = agentId;
    agentsList.appendChild(newAgent);
};

socket.onclose = () => {
    console.log('Disconnected from dashboard websocket');
};

function sendCommand() {
    const command = commandInput.value;
    if (command) {
        socket.send(command);
        commandInput.value = '';
    }
}
