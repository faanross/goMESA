import { ref, reactive } from 'vue';
import { notify } from "@kyvg/vue3-notification";

// State that will be shared across components
export const connectionState = ref('disconnected');
export const agents = reactive([]);
export const commandHistory = reactive([]);
export const notifications = reactive([]);

// WebSocket connection
let socket = null;
let reconnectInterval = null;
let lastNotificationMessage = '';
let lastNotificationTime = 0;
const NOTIFICATION_DEBOUNCE_MS = 5000; // Only show same message every 5 seconds
const listeners = new Map();
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 2; // Limit reconnect attempts

// Function to connect to the WebSocket server
export function connectWebSocket(url = 'ws://localhost:8080/ws') {
    // Don't try to reconnect if we're already connecting or connected
    if (connectionState.value === 'connecting') {
        return;
    }

    // Clear existing connection and reconnect timer
    if (socket) {
        socket.close();
        socket = null;
    }

    if (reconnectInterval) {
        clearInterval(reconnectInterval);
        reconnectInterval = null;
    }

    connectionState.value = 'connecting';

    // Create a new WebSocket connection
    socket = new WebSocket(url);

    // Connection event handlers
    socket.onopen = () => {
        connectionState.value = 'connected';
        reconnectAttempts = 0; // Reset reconnect counter on success
        addNotification('Connected to server', 'success');

        // Request initial data
        sendMessage({ type: 'getAgents' });
    };

    socket.onclose = () => {
        if (connectionState.value === 'connected') {
            // Only notify on transition from connected to disconnected
            addNotification('Disconnected from server', 'error');
        }

        connectionState.value = 'disconnected';

        // Don't auto-reconnect - wait for manual action
        reconnectAttempts = 0;
    };

    socket.onerror = (error) => {
        console.error('WebSocket error:', error);
        // Don't show error notification - let onclose handle it
        // This prevents double notifications
    };

    socket.onmessage = (event) => {
        try {
            const message = JSON.parse(event.data);
            handleMessage(message);
        } catch (error) {
            console.error('Error parsing WebSocket message:', error);
        }
    };
}

// Function to send a message to the server
export function sendMessage(message) {
    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(message));
        return true;
    } else {
        console.error('WebSocket not connected');
        // Don't notify on every failed send - prevents spamming
        return false;
    }
}

// Function to add event listener
export function addMessageListener(type, callback) {
    if (!listeners.has(type)) {
        listeners.set(type, []);
    }
    listeners.get(type).push(callback);

    // Return a function to remove the listener
    return () => {
        const typeListeners = listeners.get(type);
        const index = typeListeners.indexOf(callback);
        if (index !== -1) {
            typeListeners.splice(index, 1);
        }
    };
}

// Function to handle incoming messages
function handleMessage(message) {
    // Process message based on type
    switch (message.type) {
        case 'agentUpdate':
            updateAgents(message.agents);
            break;

        case 'commandResponse':
            addCommandToHistory(message);
            notifyCommandResult(message);
            break;

        case 'pingResponse':
            notifyPingResult(message);
            break;

        case 'killResponse':
            notifyKillResult(message);
            break;

        case 'groupResponse':
            notifyGroupResult(message);
            break;

        case 'error':
            addNotification(message.message, 'error');
            break;

        default:
            console.log('Unknown message type:', message.type);
    }

    // Notify listeners
    if (listeners.has(message.type)) {
        for (const callback of listeners.get(message.type)) {
            callback(message);
        }
    }

    // Notify 'all' listeners
    if (listeners.has('all')) {
        for (const callback of listeners.get('all')) {
            callback(message);
        }
    }
}

// Function to update agents list
function updateAgents(newAgents) {
    // Clear current agents
    agents.splice(0, agents.length);

    // Add new agents
    agents.push(...newAgents);
}

// Function to add command to history
function addCommandToHistory(commandResponse) {
    // Check if already in history to prevent duplicates
    const existingIndex = commandHistory.findIndex(cmd =>
        cmd.agentId === commandResponse.agentId &&
        cmd.command === commandResponse.command &&
        cmd.timestamp === commandResponse.timestamp
    );

    if (existingIndex === -1) {
        // Unshift to add to beginning (newest first)
        commandHistory.unshift(commandResponse);

        // Limit history size
        if (commandHistory.length > 100) {
            commandHistory.pop();
        }
    }
}

// Function to add notification with deduplication
export function addNotification(message, type = 'info', timeout = 5000) {
    // Deduplicate notifications
    const now = Date.now();
    if (message === lastNotificationMessage &&
        (now - lastNotificationTime) < NOTIFICATION_DEBOUNCE_MS) {
        return null; // Skip duplicate notification
    }

    // Update last notification tracking
    lastNotificationMessage = message;
    lastNotificationTime = now;

    // Create notification object
    const notification = {
        id: now,
        message,
        type,
        timestamp: new Date(),
    };

    // Add to notifications list
    notifications.unshift(notification);

    // Limit notifications size
    if (notifications.length > 50) {
        notifications.pop();
    }

    // Use the notify function from the library
    notify({
        title: type.charAt(0).toUpperCase() + type.slice(1),
        text: message,
        type: type,
        duration: timeout
    });

    return notification;
}

// Helper functions for notification messages
function notifyCommandResult(message) {
    if (message.success) {
        addNotification(`Command executed successfully on ${message.agentId}`, 'success');
    } else {
        addNotification(`Command failed on ${message.agentId}: ${message.error}`, 'error');
    }
}

function notifyPingResult(message) {
    if (message.success) {
        addNotification(`Ping sent to ${message.agentId}`, 'success');
    } else {
        addNotification(`Ping failed for ${message.agentId}: ${message.error}`, 'error');
    }
}

function notifyKillResult(message) {
    if (message.success) {
        addNotification(`Kill command sent to ${message.agentId}`, 'warning');
    } else {
        addNotification(`Kill command failed for ${message.agentId}: ${message.error}`, 'error');
    }
}

function notifyGroupResult(message) {
    if (message.success) {
        addNotification(`Agent ${message.agentId} assigned to group '${message.groupName}'`, 'success');
    } else {
        addNotification(`Failed to assign agent ${message.agentId} to group: ${message.error}`, 'error');
    }
}

// Command execution functions
export function executeCommand(agentId, command) {
    return sendMessage({
        type: 'executeCommand',
        agentId,
        command,
    });
}

export function pingAgent(agentId) {
    return sendMessage({
        type: 'pingAgent',
        agentId,
    });
}

export function killAgent(agentId) {
    return sendMessage({
        type: 'killAgent',
        agentId,
    });
}

export function groupAgent(agentId, groupName) {
    return sendMessage({
        type: 'groupAgent',
        agentId,
        groupName,
    });
}

// Don't automatically connect when module is imported
// Let the App.vue component handle this instead